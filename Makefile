.PHONY: run test lint fmt vet check image

run:
	go run .

test:
	go test -race -cover ./...

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .

check: vet test lint
	test -z "$$(gofmt -l .)"

image:
	docker build -t pole-chudes-bot .
