test:
	go test -race -count 1 -cover -covermode atomic -coverprofile coverage.out $(go list ./... | grep -v ./example/)
	go tool cover -html=coverage.out -o coverage.html

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.10.1 run ./...
