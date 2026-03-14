.PHONY: test test-integration test-unit vet

# Run all tests
test: vet test-unit test-integration

# Unit tests only
test-unit:
	go test -race -count=1 -timeout 60s ./pkg/...

# Integration / E2E tests
test-integration:
	go test -race -v -count=1 -timeout 120s ./tests/integration/...

# Static analysis
vet:
	go vet ./...
