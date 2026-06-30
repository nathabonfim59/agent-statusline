.PHONY: build test clean vet release release-snapshot

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

release-snapshot:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean
