BINARY=youtrack

build:
	go build -o $(BINARY) .

test:
	go test -tags integration ./...

test-integration:
	go test -tags integration -run TestIntegration -v ./test/

dist:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build -o dist/youtrack-linux-amd64 .
	GOOS=darwin  GOARCH=arm64 go build -o dist/youtrack-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build -o dist/youtrack-darwin-amd64 .
	GOOS=windows GOARCH=amd64 go build -o dist/youtrack-windows-amd64.exe .

clean:
	rm -f $(BINARY)
	rm -rf dist/

.PHONY: build test test-integration dist clean
