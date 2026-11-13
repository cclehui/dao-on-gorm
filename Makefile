.PHONY: goget build lint test test-integration

# 根模块和 examples 子模块各有一份 go.mod / go.sum
goget:
	go mod tidy
	cd examples && go mod tidy

build:
	go build ./...
	cd examples && go build ./...

lint:
	gofmt -l .
	go vet ./...
	cd examples && go vet ./...

# 纯单元测试，不依赖外部服务
test:
	go test .

# 集成测试，需要真实 MySQL + redis（见 harness/knowledge/testing.md）
test-integration:
	cd examples && go test -v ./...

# 全量检查：单元测试 + examples 编译 + 集成测试
check: lint test test-integration
