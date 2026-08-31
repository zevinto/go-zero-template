// Package apollo 提供配置中心（携程 Apollo）的冷加载接入，基于官方 SDK
// github.com/apolloconfig/agollo/v5。
//
// 设计约定（详见 README「配置中心」章节）：
//   - 冷加载：启动时拉取一次，拉取完成后随即 Close 客户端，关闭 SDK 启动的长轮询
//     goroutine，不做热更新，改配置走重启（热更新为扩展点）；
//   - key 规范：Apollo 中的 key 使用小写点分路径（如 database.host），
//     Fetch 展开为嵌套 map 后与本地 yaml 在 map 层合并（Overlay），
//     再一次性加载——避免二次加载时必填字段（如 RestConf.Name）
//     不在 Apollo 内容中而误报 not set；
//   - 多 namespace 按 Namespaces 顺序合并，后者覆盖前者；
//   - 值不做环境变量展开（Overlay 对本地 yaml 的 ${VAR} 展开与 conf.UseEnv 等价），
//     Apollo 侧的密钥仍走 env 注入；
//   - 容灾：SDK 内置本地备份配置（IsBackupConfig），Meta 拉取失败时回退本地缓存，
//     与上层 Overlay 的本地 yaml 兜底配合，避免非 dev 环境冷启动被临时网络故障打死。
//
// 使用官方 SDK 而非自实现 HTTP 拉取的原因：SDK 统一处理 Meta 服务发现、本地备份
// 缓存、Config 合法解析与命名空间语义；为下一步无缝扩展长轮询热更新铺路
// （Fetch 返回前 close 即冷加载，保留 client 不关即热更新）。
package apollo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apolloconfig/agollo/v5"
	"github.com/apolloconfig/agollo/v5/env/config"
	goYaml "go.yaml.in/yaml/v2"

	"github.com/zeromicro/go-zero/core/conf"
)

// 默认值，Conf 对应字段为零值时生效。
const (
	defaultCluster    = "default"
	defaultNamespace  = "application"
	defaultTimeoutSec = 3
)

// Conf Apollo 引导配置（来自 etc/server.yaml 的 Apollo 段）。
type Conf struct {
	AppID            string   `json:",optional"`
	Cluster          string   `json:",optional"` // 空取 default
	MetaAddr         string   `json:",optional"` // 如 http://apollo-meta.local:8080；空表示不接入
	Namespaces       []string `json:",optional"` // 空取 [application]；顺序即覆盖顺序
	TimeoutSec       int      `json:",optional"` // 单次同步超时（秒），默认 3
	BackupConfigPath string   `json:",optional"` // SDK 本地备份缓存目录；空取 os.TempDir()/apollo-backup
}

// Fetch 使用 agollo v5 SDK 按 namespace 顺序拉取配置并展开为嵌套 map
// （key 为小写点分路径的展开）。多 namespace 中相同的 key 后者覆盖前者；
// 点分 key 互相冲突时报错。返回值交给 Overlay 与本地配置合并后加载。
//
// 冷加载语义：拉取完成后即 Close 客户端以停止 SDK 启动的长轮询 goroutine；
// 若后续需要热更新，去掉 Close 并改为常驻客户端 + AddChangeListener 即可。
func Fetch(ctx context.Context, c Conf) (map[string]any, error) {
	if c.MetaAddr == "" || c.AppID == "" {
		return nil, fmt.Errorf("Apollo.MetaAddr 与 Apollo.AppID 不能为空")
	}

	namespaces := c.Namespaces
	if len(namespaces) == 0 {
		namespaces = []string{defaultNamespace}
	}

	sdkCfg := &config.AppConfig{
		AppID:             c.AppID,
		Cluster:           clusterOrDefault(c.Cluster),
		IP:                strings.TrimSuffix(c.MetaAddr, "/"),
		NamespaceName:     strings.Join(namespaces, ","), // 逗号分隔
		SyncServerTimeout: timeoutOrDefault(c.TimeoutSec),
		IsBackupConfig:    true,                  // 内置本地备份容灾
		BackupConfigPath:  backupConfigPathOf(c), // 默认落全局临时目录，避免污染可执行文件目录
		MustStart:         true,                  // 首次同步必须成功（拉不到配置视为启动失败）
	}

	client, err := agollo.StartWithConfig(func() (*config.AppConfig, error) {
		return sdkCfg, nil
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 agollo 失败: %w", err)
	}
	// 冷加载：取完即关，停掉 SDK 内部的 server list 刷新与长轮询 goroutine
	defer client.Close()

	flat := map[string]string{}
	for _, ns := range namespaces {
		cache := client.GetConfigCache(ns)
		if cache == nil {
			return nil, fmt.Errorf("拉取 namespace %q: 未获取到配置缓存", ns)
		}
		cache.Range(func(k, v any) bool {
			ks, ok1 := k.(string)
			vs, ok2 := v.(string)
			if ok1 && ok2 {
				flat[ks] = vs // 后一 namespace 覆盖前一 namespace
			}
			return true
		})
	}

	return buildNested(flat)
}

// Overlay 将 Apollo 配置覆盖到本地 yaml 文件之上并加载进 v：
// 读本地 yaml → ${VAR} 环境变量展开（与 conf.UseEnv 等价）→ MergeInto 递归合并
// → 一次性 conf 加载。Apollo 值优先，本地独有字段保留。
func Overlay(yamlFile string, remote map[string]any, v any) error {
	raw, err := os.ReadFile(yamlFile)
	if err != nil {
		return fmt.Errorf("读取本地配置: %w", err)
	}

	local := map[string]any{}
	if err := goYaml.Unmarshal([]byte(os.ExpandEnv(string(raw))), &local); err != nil {
		return fmt.Errorf("解析本地配置: %w", err)
	}

	merged := MergeInto(normalize(local), remote)

	content, err := goYaml.Marshal(merged)
	if err != nil {
		return fmt.Errorf("序列化合并配置: %w", err)
	}
	return conf.LoadFromYamlBytes(content, v)
}

// MergeInto 将 remote 递归覆盖进 local：remote 优先，同名 map 递归合并，
// local 独有字段保留。key 一律归一为小写（与 go-zero conf 的 canonical key
// 对齐），避免 "Database" 与 "database" 并存导致覆盖不确定。
func MergeInto(local map[string]any, remote map[string]any) map[string]any {
	merged := make(map[string]any, len(local)+len(remote))
	for k, v := range local {
		mk := strings.ToLower(k)
		if m, ok := v.(map[string]any); ok {
			merged[mk] = mergeCopy(m)
			continue
		}
		merged[mk] = v
	}
	for k, v := range remote {
		mk := strings.ToLower(k)
		if rm, ok := v.(map[string]any); ok {
			if lm, ok := merged[mk].(map[string]any); ok {
				merged[mk] = MergeInto(lm, rm)
				continue
			}
			merged[mk] = mergeCopy(rm)
			continue
		}
		merged[mk] = v // remote 标量覆盖 local
	}
	return merged
}

// mergeCopy 深拷贝 map，避免合并过程改动 local 的嵌套结构。
func mergeCopy(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if vm, ok := v.(map[string]any); ok {
			out[strings.ToLower(k)] = mergeCopy(vm)
			continue
		}
		out[strings.ToLower(k)] = v
	}
	return out
}

// normalize 将 goYaml（yaml.v2）解析出的嵌套结构统一转为标准 Go 类型：
// map[interface{}]interface{} → map[string]any，[]interface{} → []any。
// yaml.v2 对嵌套映射默认产出 map[interface{}]interface{}，这会让
// MergeInto 中的 map[string]any 类型断言失败而误替换整棵子树，故先归一。
func normalize(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = normValue(v)
	}
	return out
}

func normValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return normalize(x)
	case map[any]any:
		nm := make(map[string]any, len(x))
		for k, vv := range x {
			ks, _ := k.(string)
			nm[ks] = normValue(vv)
		}
		return nm
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = normValue(item)
		}
		return out
	default:
		return v
	}
}

// buildNested 将点分 key（database.params.sslmode）展开为嵌套 map。
// 标量与子树路径冲突（如同时出现 database 和 database.host）视为配置损坏，直接报错。
func buildNested(flat map[string]string) (map[string]any, error) {
	root := map[string]any{}
	for key, value := range flat {
		parts := strings.Split(key, ".")
		cur := root
		for i, part := range parts {
			if i == len(parts)-1 {
				if _, exists := cur[part]; exists {
					return nil, fmt.Errorf("配置键冲突: %q", key)
				}
				cur[part] = value
				break
			}

			next, ok := cur[part].(map[string]any)
			if !ok {
				if _, exists := cur[part]; exists {
					return nil, fmt.Errorf("配置键冲突: %q", key)
				}
				next = map[string]any{}
				cur[part] = next
			}
			cur = next
		}
	}
	return root, nil
}

func clusterOrDefault(cluster string) string {
	if cluster == "" {
		return defaultCluster
	}
	return cluster
}

func timeoutOrDefault(sec int) int {
	if sec <= 0 {
		return defaultTimeoutSec
	}
	return sec
}

// backupConfigPathOf 返回 SDCKG 本地备份配置目录；未显式配置时落到系统临时目录，
// 避免把备份文件写进可执行文件所在的部署目录。
func backupConfigPathOf(c Conf) string {
	if c.BackupConfigPath != "" {
		return c.BackupConfigPath
	}
	return filepath.Join(os.TempDir(), "apollo-backup")
}
