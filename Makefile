.DEFAULT: help

help:
	@echo "Commands:"
	@echo "  build"
	@echo "  install"

.PHONY: build
build: 
	go build -o rssy cmd/main.go


.PHONY: install
install: build
	rm ${GOPATH}/bin/rssy 2> /dev/null || true
	mv rssy ${GOPATH}/bin/