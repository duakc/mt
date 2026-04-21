.PHONY: generate
generate:
	@go generate ./...

.PHONY: test
test:
	@go test ./...

.PHONY: toolchain
toolchain:
	@asdf install

.PHONY: fmt
fmt:
	@golangci-lint fmt

.PHONY: lint
lint: fmt
	@golangci-lint run
