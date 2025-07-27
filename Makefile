.PHONY: test
test:
	@echo "run tests"
	# @go test -v -json ./... | tparse -all
	@go test $(go list ./... | grep -v /cmd/) -v -json | tparse -all

.PHONY: lint
lint:
	@echo "run lint"
	@golangci-lint run

.PHONY: dev-up
dev-up: export AWS_ACCESS_KEY_ID="admin"
dev-up: export AWS_SECRET_ACCESS_KEY="password"
dev-up: export AWS_ENDPOINT_URL="http://localhost:9000"
dev-up:
	@echo "starting dev environment with minio AWS credentials"
	@docker-compose up -d

.PHONY: dev-down
dev-down:
	@echo "stopping dev environment"
	@docker-compose down

