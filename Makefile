.PHONY: setup test vet

setup:
	go mod download

test:
	go test -v ./...

vet:
	go vet ./...
