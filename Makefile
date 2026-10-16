.DEFAULT_GOAL := build

GOTOOLCHAIN ?= go1.27.1
export GOTOOLCHAIN
STATICCHECK_VERSION ?= v0.8.1

.PHONY: install-tools format comments lint test-build test build tidy

install-tools:
	go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	python3 -m pip install --quiet --upgrade git+https://github.com/botforge-pro/commentcensor.git

format:
	gofmt -w .

comments:
	commentcensor .

lint: comments
	go vet ./...
	gofmt -l . | (! grep .)
	$(shell go env GOPATH)/bin/staticcheck ./...
	go mod tidy -diff

test-build:
	go build ./...
	go test -run '^$$' ./...

test:
	go test -race -count=1 $(or $(TEST),./...)

build: lint test-build test
	go build ./...

tidy:
	go mod tidy
