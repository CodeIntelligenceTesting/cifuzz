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

project := "code-intelligence.com/cifuzz"

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
	go install github.com/incu6us/goimports-reviser/v2@latest

.PHONY: install
install:
	go run cmd/installer/main.go

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
	find . -type f -name "*.go" -exec goimports-reviser -project-name $(project) -file-path {} \;

.PHONY: fmt/check
fmt/check:
	@DIFF=$$(find . -type f -name "*.go" -exec goimports-reviser -project-name $(project) -list-diff -file-path {} \;); \
	if [ -n "$$DIFF" ]; then \
		echo >&2 "Unformatted files:\n$$DIFF"; \
		exit 1; \
	fi;

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: tidy/check
tidy/check:
	# Replace with `go mod tidy -check` once that's available, see
	# https://github.com/golang/go/issues/27005
	if [ -n "$$(git status --porcelain go.mod go.sum)" ]; then       \
		echo >&2 "Error: The working tree has uncommitted changes."; \
		exit 1;                                                      \
	fi
	go mod tidy
	if [ -n "$$(git status --porcelain go.mod go.sum)" ]; then \
		echo >&2 "Error: Files were modified by go mod tidy";  \
		git checkout go.mod go.sum;                            \
		exit 1;                                                \
	fi

.PHONY: test
test: deps build/$(current_os)
	go test -v ./...

.PHONY: test/unit
test/unit: deps
	go test -v ./... -short

.PHONY: test/unit/concurrent
test/unit/concurrent: deps
	go test -v ./... -short -count=10

.PHONY: test/integration
test/integration: deps build/$(current_os)
	go test -v ./... -run 'TestIntegration.*'

.PHONY: test/race
test/race: deps build/$(current_os)
	go test -v ./... -race

.PHONY: test/coverage
test/coverage: deps
	go test -v ./... -coverprofile coverage.out
	go tool cover -html coverage.out

.PHONY: site/setup
site/setup:
	-rm -rf site
	git clone git@github.com:CodeIntelligenceTesting/cifuzz.wiki.git site 

.PHONY: site/generate
site/generate: deps
	rm -f ./site/*.md
	go run ./cmd/gen-docs/main.go --dir ./site/
	cp -R ./docs/*.md ./site

.PHONY: site/update
site/update:
	git -C site add -A
	git -C site commit -m "update docs" || true
	git -C site push
