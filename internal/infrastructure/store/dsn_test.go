package store

import (
	"testing"

	"github.com/zevinto/go-zero-template/internal/config"
)

func TestRuntimeDSN(t *testing.T) {
	cases := []struct {
		name    string
		conf    config.DatabaseConf
		want    string
		wantErr bool
	}{
		{
			name: "postgres 基础",
			conf: config.DatabaseConf{Adapter: "postgres", Host: "db.local", Username: "app", Password: "p@ss", Name: "order"},
			want: "postgres://app:p%40ss@db.local:5432/order",
		},
		{
			name: "postgres 带 sslmode",
			conf: config.DatabaseConf{Adapter: "postgres", Host: "db.local", Username: "app", Password: "x", Name: "order",
				Params: map[string]string{"sslmode": "disable"}},
			want: "postgres://app:x@db.local:5432/order?sslmode=disable",
		},
		{
			name: "mysql 强制 parseTime/loc，可被 Params 覆盖",
			conf: config.DatabaseConf{Adapter: "mysql", Host: "db.local", Username: "app", Password: "x", Name: "order",
				Params: map[string]string{"loc": "Asia%2FShanghai"}},
			want: "mysql://app:x@tcp(db.local:3306)/order?loc=Asia%252FShanghai&parseTime=true",
		},
		{
			name: "mysql 指定端口",
			conf: config.DatabaseConf{Adapter: "mysql", Host: "db.local", Port: 3307, Username: "app", Password: "x", Name: "order"},
			want: "mysql://app:x@tcp(db.local:3307)/order?loc=Local&parseTime=true",
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
			got, err := runtimeDSN(c.conf)
			if c.wantErr {
				if err == nil {
					t.Fatalf("runtimeDSN(%+v) error = nil, want error", c.conf)
				}
				return
			}
			if err != nil {
				t.Fatalf("runtimeDSN(%+v) error = %v", c.conf, err)
			}
			if got != c.want {
				t.Fatalf("runtimeDSN(%+v) = %q, want %q", c.conf, got, c.want)
			}
		})
	}
}

// 可选依赖语义：未配置时返回 nil 而非报错。
func TestNewDBUnconfigured(t *testing.T) {
	db, err := NewStore(config.DatabaseConf{})
	if db != nil || err != nil {
		t.Fatalf("NewDB(empty) = %v, %v; want nil, nil", db, err)
	}
}
