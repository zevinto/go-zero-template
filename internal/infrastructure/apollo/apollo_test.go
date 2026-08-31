package apollo_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/zevinto/go-zero-template/internal/config"
	"github.com/zevinto/go-zero-template/internal/infrastructure/apollo"
)

// mockConfig 模拟 agollo v5 SDK 的配置接口：
//
//	GET {meta}/services/config?...                       服务发现（列表）
//	GET {meta}/configfiles/json/{app}/{cluster}/{ns}?..  拉取配置（body 为扁平 key-value JSON）
//
// 冷加载时 SDK 只会用直接 host 拉取 configfiles，服务发现返回的列表用于容灾轮询；
// 这里一并提供，避免 StartWithConfig 启动的服务发现 goroutine 打空。
type mockApollo struct {
	srv   *httptest.Server
	paths []string
	mu    sync.Mutex
}

func newMockApollo(t *testing.T, namespaces map[string]map[string]string) *mockApollo {
	t.Helper()
	m := &mockApollo{}

	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.paths = append(m.paths, r.URL.Path)
		m.mu.Unlock()

		// 服务发现：返回 config server 列表（指向自身）
		if strings.HasPrefix(r.URL.Path, "/services/config") {
			_ = json.NewEncoder(w).Encode(map[string][]map[string]any{
				"servers": {
					{
						"appName":     "mock-apollo",
						"instanceId":  "i-0",
						"homepageUrl": strings.TrimSuffix(m.srv.URL, "/"),
						"isDown":      false,
					},
				},
			})
			return
		}

		// 拉取配置：/configfiles/json/{app}/{cluster}/{ns}
		rest := strings.TrimPrefix(r.URL.Path, "/configfiles/json/")
		parts := strings.Split(rest, "/")
		if len(parts) < 3 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ns := parts[len(parts)-1]
		kv, ok := namespaces[ns]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// processJSONFiles 直接把响应体反序列化为扁平 key-value map
		_ = json.NewEncoder(w).Encode(kv)
	}))

	t.Cleanup(m.srv.Close)
	return m
}

// url 返回 Meta 地址（去掉结尾斜杠，命中 SDK 的 GET host + suffix 拼接）。
func (m *mockApollo) url() string {
	return strings.TrimSuffix(m.srv.URL, "/")
}

func TestFetchExpandsDottedKeys(t *testing.T) {
	m := newMockApollo(t, map[string]map[string]string{
		"application": {"database.adapter": "postgres", "database.name": "order"},
	})

	got, err := apollo.Fetch(context.Background(), apollo.Conf{
		AppID: "gzt", MetaAddr: m.url(), Namespaces: []string{"application"},
	})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	db, ok := got["database"].(map[string]any)
	if !ok || db["adapter"] != "postgres" || db["name"] != "order" {
		t.Fatalf("nested map = %#v, want database.{adapter,name}", got)
	}
}

func TestFetchMergesNamespacesWithOverride(t *testing.T) {
	m := newMockApollo(t, map[string]map[string]string{
		"application": {"database.adapter": "postgres", "database.name": "order"},
		"db":          {"database.name": "order-prod"},
	})

	got, err := apollo.Fetch(context.Background(), apollo.Conf{
		AppID:      "gzt",
		MetaAddr:   m.url(),
		Namespaces: []string{"application", "db"},
	})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// application 在前、db 在后：adapter 保留、name 被覆盖
	db := got["database"].(map[string]any)
	if db["adapter"] != "postgres" || db["name"] != "order-prod" {
		t.Fatalf("merged = %#v, want adapter=postgres name=order-prod", db)
	}
}

// 点分 key 互相冲突（标量与子树路径重叠）属于配置损坏，必须报错而非静默覆盖。
func TestFetchKeyConflict(t *testing.T) {
	m := newMockApollo(t, map[string]map[string]string{
		"application": {"database": "broken", "database.host": "h"},
	})

	if _, err := apollo.Fetch(context.Background(), apollo.Conf{
		AppID: "gzt", MetaAddr: m.url(), Namespaces: []string{"application"},
	}); err == nil {
		t.Fatal("Fetch() error = nil, want key conflict error")
	}
}

func TestFetchUnconfigured(t *testing.T) {
	if _, err := apollo.Fetch(context.Background(), apollo.Conf{}); err == nil {
		t.Fatal("Fetch(empty) error = nil, want error")
	}
}

// Overlay 端到端：本地 yaml 提供必填字段（Name 等），Apollo 覆盖 database 段，
// ${VAR} 展开、大小写归一、本地独有字段保留一并验证。
func TestOverlayEndToEnd(t *testing.T) {
	m := newMockApollo(t, map[string]map[string]string{
		"application": {
			"database.host":     "db-from-apollo",
			"database.password": "from-apollo",
		},
	})

	remote, err := apollo.Fetch(context.Background(), apollo.Conf{
		AppID: "gzt", MetaAddr: m.url(), Namespaces: []string{"application"},
	})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	dir := t.TempDir()
	yamlFile := filepath.Join(dir, "server.yaml")
	local := `
Name: gzt-server
Host: 0.0.0.0
Port: 8888
Database:
  Adapter: postgres
  Host: localhost
  Password: ${DB_PASSWORD}
`
	t.Setenv("DB_PASSWORD", "env-secret")
	if err := os.WriteFile(yamlFile, []byte(local), 0o600); err != nil {
		t.Fatal(err)
	}

	var c config.Config
	if err := apollo.Overlay(yamlFile, remote, &c); err != nil {
		t.Fatalf("Overlay() error = %v", err)
	}

	// 本地必填字段保留（这是 Overlay 存在的理由——二次加载会误报 not set）
	if c.Name != "gzt-server" || c.Host != "0.0.0.0" || c.Port != 8888 {
		t.Fatalf("local required fields lost: %+v", c.RestConf)
	}
	// Apollo 覆盖生效
	if c.Database.Host != "db-from-apollo" || c.Database.Password != "from-apollo" {
		t.Fatalf("database = %+v, want values from apollo", c.Database)
	}
	// 本地独有字段保留
	if c.Database.Adapter != "postgres" {
		t.Fatalf("adapter = %q, want postgres (local only)", c.Database.Adapter)
	}
	// 被覆盖字段不再走 env 展开值
	if c.Database.Password == "env-secret" {
		t.Fatal("apollo value should override env-expanded local value")
	}
}
