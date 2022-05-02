unit_tests = $$(go list ./... | grep -v integration)

current_os :=
ifeq ($(OS),Windows_NT)
	current_os = windows
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		current_os = linux
	endif
	ifeq ($(UNAME_S),Darwin)
		current_os = darwin 
	endif
endif

build_path = build/bin/
binary_prefix = cifuzz_
int_test_prefix = int_test_

default:
	@echo cifuzz

.PHONY: clean
clean:
	rm -rf build/

.PHONY: deps
deps:
	go mod download

.PHONY: deps/dev
deps/dev: deps
	go install honnef.co/go/tools/cmd/staticcheck@latest

.PHONY: build
build: build/linux build/windows build/darwin ;

.PHONY: build/linux
build/linux: deps
	env GOOS=linux GOARCH=amd64 go build -o $(build_path)$(binary_prefix)linux cmd/cifuzz/main.go

.PHONY: build/windows
build/windows: deps
	env GOOS=windows GOARCH=amd64 go build -o $(build_path)$(binary_prefix)windows.exe cmd/cifuzz/main.go

.PHONY: build/darwin
build/darwin: deps
	env GOOS=darwin GOARCH=amd64 go build -o $(build_path)$(binary_prefix)darwin cmd/cifuzz/main.go

.PHONY: build/integration
build/integration: build/$(current_os) 
	go test -c -o $(build_path)$(int_test_prefix)$(current_os) integration/cli_test.go

.PHONY: lint
lint: deps/dev
	staticcheck ./...
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: fmt/check
fmt/check:
	if [ "$$(gofmt -d -l . | wc -l)" -gt 0 ]; then exit 1; fi;

.PHONY: test
test: test/unit test/integration;

.PHONY: test/unit
test/unit: deps
	go test $(unit_tests)

.PHONY: test/integration
test/integration: build/integration;
	cd $(build_path) && \
	./$(int_test_prefix)$(current_os)

.PHONY: test/race
test/race: deps
	go test -race $(unit_tests)

.PHONY: test/coverage
test/coverage: deps
	go test $(unit_tests) -coverprofile coverage.out
	go tool cover -html coverage.out

