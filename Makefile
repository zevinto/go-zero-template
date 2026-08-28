API_FILE := api/server.api
SERVER_DIR := cmd/server
SERVER_MAIN := $(SERVER_DIR)/main.go

.PHONY: api build run test vet lint clean

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
