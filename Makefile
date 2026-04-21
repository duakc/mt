.PHONY: test
test: lint
	@go test ./...

.PHONY: generate
generate:
	@go generate ./...

.PHONY: toolchain
toolchain:
	@asdf install

.PHONY: fmt
fmt:
	@golangci-lint fmt

.PHONY: lint
lint: fmt
	@golangci-lint run
