.PHONY: build test run clean

build:
	cd dashboard && npm run build
	rm -rf internal/api/dashboard_dist/*
	cp -r dashboard/out/. internal/api/dashboard_dist/
	go build -o bin/corvade ./cmd/corvade

test:
	go test ./... -v

run:
	go run ./cmd/corvade

clean:
	rm -rf bin/
