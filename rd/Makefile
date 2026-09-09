# rd 服务 Makefile。
# proto 契约真相源在 docs/api/v1/api.proto；生成物 rd/api/api.pb.go 入库（改 proto 即改契约）。

# protoc-gen-go 默认 go install 到 $(HOME)/go/bin
PATH := $(HOME)/go/bin:$(PATH)

.PHONY: proto migrate build vet test fmt

# proto 从真相源重新生成 Go DTO（buf + protoc-gen-go）。
proto:
	cd ../docs/api/v1 && buf generate

# migrate 编译当前代码并对配置库执行 schema 迁移（Bootstrap + Run 后退出）。
migrate:
	go run ./cmd/rd migrate

fmt:
	gofmt -l -w .

build:
	go build ./...

vet:
	go vet ./...

test:
	go test ./...
