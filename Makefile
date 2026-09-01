API_FILE := api/server.api
SERVER_DIR := cmd/server
SERVER_MAIN := $(SERVER_DIR)/main.go

.PHONY: api build run test vet lint clean migrate-create migrate-up migrate-down migrate-to migrate-version migrate-force

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
migrate-create:
	@if ! command -v migrate >/dev/null 2>&1; then \
		echo "错误: 未安装 golang-migrate CLI，请先安装:"; \
		echo "  brew install golang-migrate   # 或 go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		exit 1; \
	fi
	migrate create -ext sql -dir migrations -ts $(NAME)

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
