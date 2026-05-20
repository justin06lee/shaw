SHELL := /bin/bash
BIN := shaw
MODULE := github.com/justin06lee/shaw

.PHONY: help build install uninstall test vet fmt clean

help:
	@echo "shaw — terminal typing trainer"
	@echo
	@echo "Targets:"
	@echo "  make build      build ./$(BIN) in the current directory"
	@echo "  make install    install $(BIN) into \$$(go env GOBIN) (or \$$GOPATH/bin)"
	@echo "  make uninstall  remove the installed $(BIN) binary"
	@echo "  make test       run all tests"
	@echo "  make vet        go vet ./..."
	@echo "  make fmt        gofmt -w ."
	@echo "  make clean      remove the local build artifact"

build:
	go build -o $(BIN) .

install:
	go install ./...
	@echo
	@echo "installed $(BIN) to $$( [ -n "$$(go env GOBIN)" ] && go env GOBIN || echo $$(go env GOPATH)/bin )"
	@echo "make sure that directory is on your PATH"

uninstall:
	rm -f "$$( [ -n "$$(go env GOBIN)" ] && go env GOBIN || echo $$(go env GOPATH)/bin )/$(BIN)"

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -f $(BIN)
