COMPONENTS ?= packing-cli packing-server
DOCKER_USERNAME ?= glynhanmer

include ./dubplate.Makefile
include ./go.Makefile
include ./docker.Makefile

.PHONY: frontend
frontend:
	${MAKE} -C frontend elm-js
