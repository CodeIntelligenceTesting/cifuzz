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

binary_base_path = build/bin/cifuzz_

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
	env GOOS=linux GOARCH=amd64 go build -o $(binary_base_path)linux cmd/cifuzz/main.go

.PHONY: build/windows
build/windows: deps
	env GOOS=windows GOARCH=amd64 go build -o $(binary_base_path)windows.exe cmd/cifuzz/main.go

.PHONY: build/darwin
build/darwin: deps
	env GOOS=darwin GOARCH=amd64 go build -o $(binary_base_path)darwin cmd/cifuzz/main.go

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
test: deps build/$(current_os)
	go test ./...

.PHONY: test/unit
test/unit: deps
	go test ./... -short

.PHONY: test/unit/concurrent
test/unit/concurrent: deps
	go test ./... -short -count=10 

.PHONY: test/integration
test/integration: deps build/$(current_os)
	go test ./... -run 'TestIntegration.*'

.PHONY: test/race
test/race: deps build/$(current_os)
	go test ./... -race

.PHONY: test/coverage
test/coverage: deps
	go test ./... -coverprofile coverage.out
	go tool cover -html coverage.out

