NAME=conoha-net
BINDIR=bin
GOARCH=amd64
GOFLAGS=-ldflags '-s -w'

all: clean linux darwin windows

windows:
	GOOS=$@ GOARCH=$(GOARCH) CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINDIR)/$@/$(NAME).exe
	cd bin/$@; zip $(NAME).$(GOARCH).zip $(NAME).exe

darwin:
	GOOS=$@ GOARCH=$(GOARCH) CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINDIR)/$@/$(NAME)
	cd bin/$@; gzip -c $(NAME) > $(NAME)-osx.$(GOARCH).gz

linux:
	GOOS=$@ GOARCH=$(GOARCH) CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINDIR)/$@/$(NAME)
	cd bin/$@; gzip -c $(NAME) > $(NAME)-linux.$(GOARCH).gz

clean:
	rm -rf $(BINDIR)/*

test:
	go test -v github.com/hironobu-s/conoha-net/...
