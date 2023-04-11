dev: build vet lint

ci: env verify build vet

env:
	go env

verify:
	go mod verify

build:
	go build

vet:
	go vet

lint:
	which golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

fieldalignment:
	@which fieldalignment || go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
	fieldalignment -test=false ./...
