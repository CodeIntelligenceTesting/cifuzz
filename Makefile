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

binary_base_path = build/bin/cifuzz

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
	go install github.com/incu6us/goimports-reviser/v2@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.46.2

.PHONY: install
install:
	go run cmd/bundler/main.go --version dev && go run -tags installer cmd/installer/main.go && rm -r installer-bundle/bundle

.PHONY: installer
installer:
	go run cmd/bundler/main.go --version dev && go build -tags installer -o build/bin/installer cmd/installer/main.go && rm -r installer-bundle/bundle

.PHONY: build
build: build/$(current_os)

.PHONY: build/all
build/all: build/linux build/windows build/darwin ;

.PHONY: build/linux
build/linux: deps
	env GOOS=linux GOARCH=amd64 go build -o $(binary_base_path)_linux cmd/cifuzz/main.go

.PHONY: build/windows
build/windows: deps
	env GOOS=windows GOARCH=amd64 go build -o $(binary_base_path)_windows.exe cmd/cifuzz/main.go

.PHONY: build/darwin
build/darwin: deps
	env GOOS=darwin GOARCH=amd64 go build -o $(binary_base_path)_darwin cmd/cifuzz/main.go

.PHONY: lint
lint: deps/dev
	golangci-lint run

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

.PHONY: test/integration
test/integration: deps
	go test -v ./... -run 'TestIntegration.*'

.PHONY: test/integration/sequential
test/integration/sequential: deps
	go test -v -parallel=1 ./... -run 'TestIntegration.*'

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
