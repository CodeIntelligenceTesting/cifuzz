# cifuzz

## Getting started
To initialize our project with cifuzz just execute `cifuzz init` in the root directory of your project. This will create
a file named `cifuzz.yaml` containing the needed configuration.

## Setup

### Use with `go get`

Since the repository is currently private, `go get code-intelligence.com/cifuzz` requires the following one-time setup:

```
$ printf '[url "ssh://git@github.com/CodeIntelligenceTesting/cifuzz"]\n\tinsteadOf = https://github.com/CodeIntelligenceTesting/cifuzz\n' >> ~/.gitconfig
$ go env -w GOPRIVATE=code-intelligence.com/*
```
