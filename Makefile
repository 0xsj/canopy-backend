.PHONY: fmt test test-verbose test-coverage

fmt:
	go fmt ./...

test:
	go test ./...

test-verbose:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
