.PHONY: build test clean vet

BINARY := claude-statusline

build:
	go build -o $(BINARY) .

vet:
	go vet ./...

$(BINARY): build

test: build vet
	@bash test.sh

clean:
	rm -f $(BINARY)
