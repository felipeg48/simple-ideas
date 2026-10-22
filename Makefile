BIN := bin/si

.PHONY: build test install play-example examples check fmt clean

build:
	go build -o $(BIN) ./cmd/si

test:
	gofmt -l . | (! grep .)
	go vet ./...
	go test ./...

install:
	go install ./cmd/si

play-example: build
	$(BIN) play examples --open

examples: build
	$(BIN) --output out/examples examples

check: build
	$(BIN) check examples

fmt:
	gofmt -w .

clean:
	rm -rf bin out html
