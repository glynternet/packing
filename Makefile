COMPONENTS ?= packing-cli packing-server
DOCKER_USERNAME ?= glynhanmer

include ./dubplate.Makefile
include ./go.Makefile
include ./docker.Makefile

.PHONY: frontend setup-frontend test test-browser check
frontend:
	${MAKE} -C frontend elm-js

setup-frontend:
	$(MAKE) -C frontend setup

test:
	go test ./...
	$(MAKE) -C frontend test

test-browser:
	$(MAKE) -C frontend test-browser

check:
	$(MAKE) -C frontend check-format
	$(MAKE) test
	$(MAKE) test-browser
