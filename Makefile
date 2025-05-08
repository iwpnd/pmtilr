.PHONY: test
test:
	@echo "run tests"
	@go test -v -json ./... | tparse -all

.PHONY: lint
lint:
	@echo "run lint"
	@golangci-lint run

