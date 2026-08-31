// Package cache 提供 Redis 连接工厂与缓存相关二次封装（分布式锁、幂等键、限流器等）。
// 连接实例由 svc 装配持有，logic 层经 ServiceContext 使用。
//
// 包名与 go-zero 的 core/stores/cache 重名：同一文件同时引用两者时，
// 需为其中一方加 import 别名。
package cache

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// NewRedis 创建 Redis 客户端。Redis.Host 未配置时返回 (nil, nil)——可选依赖。
// 注：RedisKeyConf 是 zRPC 服务端鉴权专用配置（Key 为鉴权 hash 的 key 名），
// REST 服务不需要。
func NewRedis(c redis.RedisConf) (*redis.Redis, error) {
	if c.Host == "" {
		return nil, nil
	}
	return redis.NewRedis(c)
}
