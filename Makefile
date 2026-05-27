BINARY  := lrclib
CMD     := ./cmd/lrclib

.PHONY: build run test test-int lint fmt clean release-dry

build:
	CGO_ENABLED=0 go build -o $(BINARY) $(CMD)

run:
	CGO_ENABLED=0 go run $(CMD)

test:
	CGO_ENABLED=0 go test ./...

test-int:
	CGO_ENABLED=0 go test -tags=integration ./...

lint:
	golangci-lint run

fmt:
	gofumpt -w .

clean:
	rm -rf dist/ tmp/ $(BINARY) $(BINARY).exe

release-dry:
	goreleaser release --snapshot --clean
