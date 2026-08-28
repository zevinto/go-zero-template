# go-zero-template

go-zero 单体服务模板：统一的目录结构、统一响应格式 `{code, message, data}` 与业务错误码约定。

## 环境要求

- Go >= 1.26
- [goctl](https://go-zero.dev/docs/tasks-installation) >= 1.10（仅生成代码时需要）

## 初始化新项目

复制本目录后执行（脚本会自动识别当前模块名并完成替换）：

```bash
./scripts/init.sh github.com/your-org/order-service
```

## 目录结构

```
api/                 # API 定义（.api 文件）
├── server.api       # 服务入口：路由与分组定义
└── base.api         # 公共类型：分页元信息等
cmd/server/main.go   # 启动入口：加载配置、安装响应包装、注册路由
etc/server.yaml      # 配置文件
internal/
├── config/          # 配置结构体
├── handler/         # HTTP 层（goctl 生成，不含业务）
├── logic/           # 业务逻辑层（写在这里）
├── svc/             # ServiceContext：依赖注入点（DB/Redis 客户端挂这里）
├── types/           # 请求/响应类型（goctl 生成）
├── xerror/          # 业务错误码与错误类型
├── xvalidator/      # 参数校验（go-playground/validator 包装）
└── xresponse/       # 统一响应包装（安装到 httpx）
```

## 常用命令

| 命令        | 说明                              |
| ----------- | --------------------------------- |
| make api    | 根据 api/*.api 重新生成代码       |
| make run    | 本地启动（默认 etc/server.yaml）  |
| make build  | 编译到 bin/server                 |
| make test   | 运行测试                          |
| make vet    | go vet 静态检查                   |
| make lint   | golangci-lint                     |
| make clean  | 清理构建产物                      |

## 新增一个接口

1. 编辑 `api/server.api`，声明类型与路由：

   ```api
   type GetUserRequest {
       Id int64 `path:"id"`
   }

   @server (
       group: user
   )
   service server {
       @handler getUser
       get /user/:id (GetUserRequest) returns (GetUserResponse)
   }
   ```

2. 执行 `make api` 生成代码；
3. 在生成的 `internal/logic/user/get_user_logic.go` 中填写业务逻辑。

注意：handler 层由 goctl 生成，已有的 handler/logic 文件不会被覆盖，但请勿在其中写业务；同时清理不再使用的旧 group 目录（goctl 不会删除它们）。

## 统一响应格式

成功响应由 `xresponse.Setup()` 安装的包装器自动处理：

```json
{"code": 0, "message": "success", "data": {"message": "pong"}}
```

业务错误在 logic 层直接返回 `*xerror.CodeError`，转成 HTTP 200 + 对应 code：

```go
return nil, xerror.ErrNotFound                                             // 预定义哨兵错误
return nil, xerror.Validationf("收货地址超出配送范围")                      // 带上下文文案
return nil, xerror.Wrap(err, xerror.CodeInternal, "订单保存失败，请稍后重试") // 包装底层 err，详情进日志
// → {"code": 42200, "message": "收货地址超出配送范围", "data": {}}
```

所有接口的 HTTP 状态码一律返回 200，客户端以响应体中的 code 判断结果。无法归类的错误分两种兜底：参数解析失败（go-zero `httpx.Parse` 抛出的错误）返回 40000 + 固定文案"请求参数错误"；其余按内部错误处理，返回 50000 + 固定文案"系统繁忙"。两条路径都不透传原始错误信息，完整内容写入日志排查。

### 错误码约定

| 分段        | 含义                                          |
| ----------- | --------------------------------------------- |
| 0           | 成功                                          |
| 10000–39999 | 业务模块专属段：每模块占用一整千位区间        |
| 40000–49999 | 客户端类：前三位镜像 HTTP 类别，末两位为顺序号 |
| 50000–59999 | 服务端类                                      |

常用客户端码：40000 参数错误、40100 未登录、40101 凭证无效、40300 无权限、40400 资源不存在、40900 冲突、42200 业务校验失败、42900 触发限流。

新模块接入时先在 `internal/xerror/errors.go` 登记自己的千位区间再定义错误码，禁止在使用处硬编码数字。

## 参数校验

参数校验分两层，规则重叠时优先用 go-zero 原生标签：

1. **go-zero 标签**（`optional` / `default` / `range` / `options`）：写在来源标签值后面（如 `form:"age,range=[0:150]"`），`httpx.Parse` 阶段执行，失败返回 40000；
2. **validate 标签**（go-playground/validator）：负责格式与复杂规则（`email`、`min`、跨字段等），由 `internal/xvalidator` 包装并在 `main.go` 挂载（`httpx.SetValidator(xvalidator.New())`），失败同样返回 40000 + 可读文案，如 `"参数校验失败: Email 邮箱格式不正确"`。

validate 标签直接写在 `.api` 文件的类型字段上，goctl 会把它原样生成到 `internal/types`（无需自定义模板）：

```api
type RegisterRequest {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=6"`
}
```

新增自定义规则：在 `internal/xvalidator/validator.go` 中 `v.RegisterValidation(...)` 注册，并在 `tagDesc` 补充可读文案。

## 测试示例

参考两个范式：

- logic 层单测：`internal/logic/health/ping_logic_test.go`
- handler 层集成测试（含统一响应格式断言）：`internal/handler/health/ping_handler_test.go`
