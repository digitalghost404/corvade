.PHONY: build test run clean

build:
	go build -o bin/corvade ./cmd/corvade

test:
	go test ./... -v

run:
	go run ./cmd/corvade

clean:
	rm -rf bin/
