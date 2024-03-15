.PHONY: generate build lint test test-ci clean

PACKAGES = $(go list ./...)
TEST_FLAGS ?= -v

build: generate lint
	go build $(PACKAGES)

generate:
	go get .
	go generate $(PACKAGES)

lint:
	go get .
	go run honnef.co/go/tools/cmd/staticcheck --tags=testing $(PACKAGES)
	go vet --tags=testing $(PACKAGES)

test: build
	go test --tags=testing $(TEST_FLAGS) $(PACKAGES) ./...

test-ci: build
	go test --tags=testing $(TEST_FLAGS) $(PACKAGES) ./...  2>&1 | go run github.com/jstemmer/go-junit-report/v2 -set-exit-code > test-report.xml

clean:
	find . -name 'ignite-config-*.xml' -delete
	find . -name 'log4j-*.xml' -delete
	find . -name 'ignite-log-*.txt' -delete
	find . -name 'test-report.xml' -delete

