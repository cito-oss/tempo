test:
	go test -race -count 1 -cover -covermode atomic -coverprofile coverage.out $(go list ./... | grep -v ./example/)
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...
