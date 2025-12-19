.PHONY: setup test vet

setup:
	go mod download

test:
	go test -v ./copilot/...

vet:
	go vet ./copilot/...
