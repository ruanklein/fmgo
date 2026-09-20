GO ?= go

.PHONY: help test vet examples check

help:
	@printf '%s\n' \
		'make test     Run the Go test suite.' \
		'make vet      Run go vet.' \
		'make examples Build integration examples into examples/.' \
		'make check    Run tests, vet, and build examples.'

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

examples:
	mkdir -p examples
	$(GO) build -o examples/respond ./cmd/respond
	$(GO) build -o examples/server ./cmd/server
	$(GO) build -o examples/stream ./cmd/stream
	$(GO) build -o examples/structured ./cmd/structured

check: test vet examples
