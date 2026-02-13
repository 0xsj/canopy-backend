CONTEXTS := identity discussion seed deliverable synthesis session \
            notification workspace organization convergence exploration ledger

.PHONY: fmt test test-verbose test-coverage sqlc-generate

fmt:
	go fmt ./...

test:
	go test ./...

test-verbose:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

sqlc-generate:
	@for ctx in $(CONTEXTS); do \
		echo "sqlc generate: $$ctx"; \
		pushd internal/$$ctx/adapter/postgres > /dev/null && sqlc generate && popd > /dev/null; \
	done
