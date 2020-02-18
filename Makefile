BINARIES ?= packing-cli packing-server
DOCKER_USERNAME ?= glynhanmer

include ./dubplate.Makefile
include ./go.Makefile
include ./docker.Makefile
