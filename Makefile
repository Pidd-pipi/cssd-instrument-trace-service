.PHONY: build vet fmt test race run docker-build docker-run clean

GO ?= go
BINARY := cssd-instrument-trace-service

build:
	$(GO) build ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "以下文件未 gofmt:"; gofmt -l .; exit 1)

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

run:
	PORT=$${PORT:-8080} $(GO) run .

docker-build:
	docker build -t cssd-instrument-trace-service:latest .

docker-run:
	docker run --rm -p $${PORT:-8080}:8080 -e PORT=8080 -v "$${PWD}/data:/app/data" cssd-instrument-trace-service:latest

clean:
	rm -f $(BINARY)
	rm -rf data/
