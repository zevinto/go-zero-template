// 运行时 DSN 拼装。
//
// 与迁移 DSN（infrastructure/migrate/adapters.go 的 SourceURL）格式不同，不共用：
//   - postgres：迁移走 pgx5://（golang-migrate 注册的 scheme），运行时走 pgx
//     stdlib 驱动的 postgres://；
//   - mysql：两者同为 go-sql-driver DSN，但运行时默认强制 parseTime=true /
//     loc=Local（DATETIME 扫描进 time.Time 的前提），Params 可覆盖。
package store

import (
	"fmt"
	"net/url"

	"github.com/zevinto/go-zero-template/internal/config"
)

// database/sql 驱动名（由本包 import 的驱动包注册）。
const (
	driverPostgres = "pgx"   // github.com/jackc/pgx/v5/stdlib
	driverMySQL    = "mysql" // github.com/go-sql-driver/mysql
)

// mysql 运行时默认参数，Params 中的同名键可覆盖。
var mysqlDefaultParams = map[string]string{
	"parseTime": "true",
	"loc":       "Local",
}

// runtimeDSN 将 DatabaseConf 拼装为 database/sql 的运行时 DSN。
func runtimeDSN(c config.DatabaseConf) (string, error) {
	if c.Host == "" || c.Name == "" {
		return "", fmt.Errorf("Database.Host 与 Database.Name 不能为空")
	}

	switch c.Adapter {
	case "postgres":
		u := url.URL{
			Scheme: "postgres",
			Host:   fmt.Sprintf("%s:%d", c.Host, portOrDefault(c.Port, 5432)),
			Path:   "/" + c.Name,
		}
		if c.Username != "" || c.Password != "" {
			u.User = url.UserPassword(c.Username, c.Password)
		}
		u.RawQuery = encodeParams(c.Params)
		return u.String(), nil

	case "mysql":
		u := url.URL{
			Scheme: "mysql",
			Host:   fmt.Sprintf("tcp(%s:%d)", c.Host, portOrDefault(c.Port, 3306)),
			Path:   "/" + c.Name,
		}
		if c.Username != "" || c.Password != "" {
			u.User = url.UserPassword(c.Username, c.Password)
		}
		params := make(map[string]string, len(c.Params)+len(mysqlDefaultParams))
		for k, v := range mysqlDefaultParams {
			params[k] = v
		}
		for k, v := range c.Params {
			params[k] = v
		}
		u.RawQuery = encodeParams(params)
		return u.String(), nil

	default:
		return "", fmt.Errorf("不支持的 Database.Adapter %q（可选: postgres, mysql）", c.Adapter)
	}
}

func portOrDefault(port, def int) int {
	if port == 0 {
		return def
	}
	return port
}

// encodeParams 键值对转 query（url.Values.Encode 保证输出键有序）。
func encodeParams(params map[string]string) string {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	return q.Encode()
}
