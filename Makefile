BIN := bin/hyprtk-usb
PKG := ./cmd/hyprtk-usb

.PHONY: build test vet clean install

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(BIN) $(PKG)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf bin

install: build
	install -Dm755 $(BIN) $(HOME)/.local/bin/hyprtk-usb
