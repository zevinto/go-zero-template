// 各数据库 adapter 的连接配置到迁移 DSN 的转换。
// DSN 形式由 golang-migrate 驱动决定（postgres → pgx5://，mysql → mysql://），
// 新增 adapter 时：import 对应驱动包 + 在 switch 中补充分支 + 在 sourceURL 补充拼装。
package migrate

import (
	"fmt"
	"net/url"

	"github.com/zevinto/go-zero-template/internal/config"
)

// Adapter 默认端口。
const (
	adapterPostgres = "postgres"
	adapterMySQL    = "mysql"
)

// SourceURL 将 DatabaseConf 拼装为对应 adapter 的迁移 DSN。
// 经 net/url 构造，用户名/密码中的特殊字符会被正确转义。
//
// 注意两个驱动的地址格式不同：
//   - postgres：标准 URL，pgx5://host:port/db；
//   - mysql：golang-migrate 会把 scheme 后的内容交给 go-sql-driver 的
//     ParseDSN 解析，TCP 地址必须是 tcp(host:port) 形式（实测验证，
//     纯 host:port 会报 "default addr for network ... unknown"）。
func SourceURL(c config.DatabaseConf) (string, error) {
	var scheme string
	switch c.Adapter {
	case adapterPostgres:
		scheme = "pgx5"
	case adapterMySQL:
		scheme = adapterMySQL
	default:
		return "", fmt.Errorf("不支持的 Database.Adapter %q（可选: postgres, mysql）", c.Adapter)
	}

	if c.Host == "" || c.Name == "" {
		return "", fmt.Errorf("Database.Host 与 Database.Name 不能为空")
	}
	if c.Port == 0 {
		if c.Adapter == adapterPostgres {
			c.Port = 5432
		} else {
			c.Port = 3306
		}
	}

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	if c.Adapter == adapterMySQL {
		addr = fmt.Sprintf("tcp(%s)", addr)
	}

	u := url.URL{
		Scheme: scheme,
		Host:   addr,
		Path:   "/" + c.Name,
	}
	if c.Username != "" || c.Password != "" {
		u.User = url.UserPassword(c.Username, c.Password)
	}
	if len(c.Params) > 0 {
		q := url.Values{}
		for k, v := range c.Params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}
