.PHONY: generate build test

PACKAGES = $(go list ./...)
TEST_FLAGS ?= -v

build: generate
	go build $(PACKAGES)

generate:
	go generate $(PACKAGES)
	go fmt $(PACKAGES)

test: build
	go test --tags=testing $(TEST_FLAGS) $(PACKAGES) ./...