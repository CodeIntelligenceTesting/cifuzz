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
test_fuzz_targets_path = testdata

default:
	@echo cifuzz

.PHONY: clean
clean:
	rm -rf build/
	make -C $(test_fuzz_targets_path) clean

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

.PHONY: build/test/fuzz-targets
build/test/fuzz-targets:
	make -C $(test_fuzz_targets_path) all

.PHONY: lint
lint: deps/dev
	staticcheck ./...
	go vet ./...

.PHONY: fmt
fmt:
	goimports -w -local code-intelligence.com .

.PHONY: fmt/check
fmt/check:
	if [ -n "$$(goimports -l -local code-intelligence.com .)" ]; then exit 1; fi;

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
test: deps build/$(current_os) build/test/fuzz-targets
	go test ./...

.PHONY: test/unit
test/unit: deps
	go test ./... -short

.PHONY: test/unit/concurrent
test/unit/concurrent: deps
	go test ./... -short -count=10 

.PHONY: test/integration
test/integration: deps build/$(current_os) build/test/fuzz-targets
	go test ./... -run 'TestIntegration.*'

.PHONY: test/race
test/race: deps build/$(current_os)
	go test ./... -race

.PHONY: test/coverage
test/coverage: deps
	go test ./... -coverprofile coverage.out
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
