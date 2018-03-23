PREFIX = /usr/local
BINDIR = $(PREFIX)/bin
DISTDIR = releases
GO_FLAGS = -ldflags="-s -w"
DEVTOOL_DIR = $(CURDIR)/devtool
GOX = $(DEVTOOL_DIR)/bin/gox
OSARCH = linux/amd64 linux/arm darwin/amd64 windows/386 windows/amd64
DIST_FORMAT = $(DISTDIR)/{{.Dir}}-{{.OS}}-{{.Arch}}

.PHONY: all test build install clean dist dist-clean

all: build

build: git-cgi-server

git-cgi-server: *.go
	go build $(GO_FLAGS)

test:
	go test $(shell go list ./... | grep -v "/vendor/")

install: all
	install -d $(BINDIR)
	install git-cgi-server $(BINDIR)

clean:
	rm -f git-cgi-server

dist: $(DEVTOOL_DIR)/bin/gox
	$(GOX) -osarch="$(OSARCH)" $(GO_FLAGS) -output="$(DIST_FORMAT)" .

$(DEVTOOL_DIR)/bin/gox:
	mkdir -p $(DEVTOOL_DIR)/{bin,pkg,src}
	GOPATH=$(DEVTOOL_DIR) go get github.com/mitchellh/gox

dist-clean:
	rm -rf git-cgi-server $(DISTDIR) $(DEVTOOL_DIR)
