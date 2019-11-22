GOOS ?= $(shell go env GOHOSTOS)
GOARCH ?= $(shell go env GOHOSTARCH)
GOFLAGS ?=
GOPROXY ?= $(shell go env GOPROXY)

.PHONY: $(notdir $(abspath $(wildcard cmd/*/)))
$(notdir $(abspath $(wildcard cmd/*/))):
	go build -o bin/$@.${GOOS} ./cmd/$@

.PHONY: clean
clean:
	rm -rf bin