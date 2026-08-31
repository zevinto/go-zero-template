package migrate

import (
	"strings"
	"testing"

	"github.com/zevinto/go-zero-template/internal/config"
)

func TestSourceURL(t *testing.T) {
	cases := []struct {
		name    string
		conf    config.DatabaseConf
		want    string
		wantErr bool
	}{
		{
			name: "postgres 默认端口",
			conf: config.DatabaseConf{Adapter: "postgres", Host: "db.local", Username: "app", Password: "p@ss", Name: "order"},
			want: "pgx5://app:p%40ss@db.local:5432/order",
		},
		{
			name: "postgres 指定端口",
			conf: config.DatabaseConf{Adapter: "postgres", Host: "db.local", Port: 6543, Username: "app", Password: "x", Name: "order"},
			want: "pgx5://app:x@db.local:6543/order",
		},
		{
			name: "mysql 默认端口（驱动要求 tcp() 包裹）",
			conf: config.DatabaseConf{Adapter: "mysql", Host: "db.local", Username: "app", Password: "x", Name: "order"},
			want: "mysql://app:x@tcp(db.local:3306)/order",
		},
		{
			name: "postgres 带 Params",
			conf: config.DatabaseConf{Adapter: "postgres", Host: "db.local", Username: "app", Password: "x", Name: "order",
				Params: map[string]string{"sslmode": "disable"}},
			want: "pgx5://app:x@db.local:5432/order?sslmode=disable",
		},
		{
			name:    "未知 adapter",
			conf:    config.DatabaseConf{Adapter: "oracle", Host: "db.local", Name: "order"},
			wantErr: true,
		},
		{
			name:    "缺少库名",
			conf:    config.DatabaseConf{Adapter: "postgres", Host: "db.local"},
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := SourceURL(c.conf)
			if c.wantErr {
				if err == nil {
					t.Fatalf("SourceURL(%+v) error = nil, want error", c.conf)
				}
				return
			}
			if err != nil {
				t.Fatalf("SourceURL(%+v) error = %v", c.conf, err)
			}
			if got != c.want {
				t.Fatalf("SourceURL(%+v) = %q, want %q", c.conf, got, c.want)
			}
		})
	}
}

// 密码中的特殊字符必须被转义，否则 URL 解析会错位。
func TestSourceURLEscapesPassword(t *testing.T) {
	got, err := SourceURL(config.DatabaseConf{
		Adapter: "postgres", Host: "h", Username: "u", Password: "p@ss:word/100%", Name: "db",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "pgx5://u:") || strings.Contains(got[len("pgx5://u:"):], "@ss") {
		t.Fatalf("password not escaped: %q", got)
	}
}
