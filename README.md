# cifuzz

## Setup

### Use with `go get`

Since the repository is currently private, `go get code-intelligence.com/cifuzz` requires the following one-time setup:

```
$ printf '[url "ssh://git@github.com/CodeIntelligenceTesting/cifuzz"]\n\tinsteadOf = https://github.com/CodeIntelligenceTesting/cifuzz\n' >> ~/.gitconfig
$ go env -w GOPRIVATE=code-intelligence.com/*
```
