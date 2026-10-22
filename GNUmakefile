default: build

.PHONY: build
build:
	go build ./...

.PHONY: install
install:
	go install .

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: generate
generate:
	go generate ./...

.PHONY: test
test:
	go test ./...

# Runs the TestAcc* tests against the Dokku server the DOKKU_* environment
# variables name.
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./internal/provider -run TestAcc -v -timeout 60m
