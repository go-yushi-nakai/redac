GO_FILES:=$(shell find . -type f -name '*.go' -print)
VERSION := $(shell git describe --tags --abbrev=0)
REVISION := $(shell git rev-parse --short HEAD)
LDFLAGS := -X 'github.com/go-yushi-nakai/redac.version=$(VERSION)' \
           -X 'github.com/go-yushi-nakai/redac.revision=$(REVISION)'

all: redac redac-acl redac-util

redac: $(GO_FILES)
	go build -ldflags "$(LDFLAGS)" -o redac ./cli/redac

redac-acl: $(GO_FILES)
	go build -ldflags "$(LDFLAGS)" -o redac-acl ./cli/redac-acl

redac-util: $(GO_FILES)
	go build -ldflags "$(LDFLAGS)" -o redac-util ./cli/redac-util

clean:
	rm -rf redac redac-acl redac-util

.PHONY: clean
