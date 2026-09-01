API_FILE := api/server.api
SERVER_DIR := cmd/server
SERVER_MAIN := $(SERVER_DIR)/main.go

# goctl model 生成数据源默认连接串（本地 postgres app 库）。
# 改库请在命令行覆盖：make gen-model ... DSN="postgres://user:pwd@host:port/db?sslmode=disable"
GEN_MODEL_DSN := postgres://postgres:$(if $(DB_PASSWORD),$(DB_PASSWORD),postgres)@localhost:5432/app?sslmode=disable

.PHONY: api build run test vet lint clean gen-model migrate-create migrate-up migrate-down migrate-to migrate-version migrate-force

api:
	@echo "Generating API code from $(API_FILE)"
	@goctl api go -api $(API_FILE) -dir . --style=go_zero
	@mkdir -p $(SERVER_DIR)
	@if [ -f "$(SERVER_MAIN)" ]; then \
		echo "$(SERVER_MAIN) already exists, removing generated server.go"; \
		rm -f server.go; \
	else \
		echo "Moving generated server.go to $(SERVER_MAIN)"; \
		mv server.go "$(SERVER_MAIN)"; \
	fi

build: ## 编译服务到 bin/
	go build -o bin/server ./cmd/server

run: ## 本地启动（默认配置 etc/server.yaml）
	go run ./cmd/server -f etc/server.yaml

test: ## 运行测试
	go test ./...

vet: ## go vet 静态检查
	go vet ./...

lint: ## golangci-lint（需自行安装）
	golangci-lint run

clean: ## 清理构建产物
	rm -rf bin

# 用法: make migrate-create NAME=add_order_status  用官方 golang-migrate CLI 生成时间戳命名的迁移
# 前提：已安装 golang-migrate CLI（brew install golang-migrate 或 go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest）
# 注意：新版 CLI 默认即时间戳命名（-format 20060102150405），无需 -seq/-ts
migrate-create:
	@if ! command -v migrate >/dev/null 2>&1; then \
		echo "错误: 未安装 golang-migrate CLI，请先安装:"; \
		echo "  brew install golang-migrate   # 或 go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		exit 1; \
	fi
	migrate create -ext sql -dir migrations $(NAME)

# 用法: make gen-model TABLE=users  基于现有表生成 model（需先跑迁移建表）
#   - TABLE 支持逗号分隔多表，每表各生成到独立子目录：
#       TABLE=users,orders           → internal/model/users/ + internal/model/orders/
#   - DIR 作为公共根目录（可选）：
#       TABLE=users,orders DIR=internal/model  → internal/model/users/ + internal/model/orders/
#   - 单表时不传 DIR，默认 internal/model/<表名>
# 数据源连接串用 DSN 变量覆盖（默认本地 app 库，postgres)；需提前安装 goctl（参见 README 环境要求）
GEN_MODEL_ROOT = $(if $(strip $(DIR)),$(DIR),internal/model)
gen-model:
	@if ! command -v goctl >/dev/null 2>&1; then \
		echo "错误: 未安装 goctl，请先安装（见 README 环境要求）"; \
		exit 1; \
	fi
	@if [ -z "$(TABLE)" ]; then \
		echo "用法: make gen-model TABLE=表1[,表2,...] [DIR=目标根目录]"; \
		echo "示例: make gen-model TABLE=users                # 生成到 internal/model/users"; \
		echo "      make gen-model TABLE=users,orders         # 生成到 internal/model/users + /orders"; \
		echo "      make gen-model TABLE=users DIR=x/model    # 生成到 x/model/users"; \
		exit 1; \
	fi
	@set -e; for t in `echo "$(TABLE)" | tr ',' ' '`; do \
		echo "生成 model: $$t -> $(GEN_MODEL_ROOT)/$$t"; \
		goctl model pg datasource --url "$(GEN_MODEL_DSN)" -s public -t "$$t" -d "$(GEN_MODEL_ROOT)/$$t" --style=go_zero; \
	done

migrate-up: ## 应用全部待执行迁移（需配置 Database.Source）
	go run ./cmd/migrate -f etc/server.yaml up

migrate-down: ## 回滚 1 个版本（改 --steps 回滚多个）
	go run ./cmd/migrate -f etc/server.yaml down --steps 1

migrate-version: ## 查看当前 schema 版本与 dirty 状态
	go run ./cmd/migrate -f etc/server.yaml version

# 用法: make migrate-to V=3  精确迁移到指定版本（可上可下）
migrate-to:
	go run ./cmd/migrate -f etc/server.yaml migrate $(V)

# 用法: make migrate-force V=3  修复 dirty 状态，强制改写版本号（V=-1 清空版本记录）
migrate-force:
	go run ./cmd/migrate -f etc/server.yaml force $(V)

# 不提供 migrate-drop / migrate-run：
# - drop 会清空所有表，属生产核弹，不暴露到命令门，避免误触；
#   确需清库时用 DBA 手动 SQL 或结合迁移文件精确倒推。
# - run 对任意 SQL 片段执行、不记版本，会破坏 dirty/版本语义，无需作为命令引入。
