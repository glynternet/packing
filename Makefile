BINARIES ?= packing-cli packing-server

include ./dubplate.Makefile
include ./go.Makefile

packing:
	$(MAKE) cmd-all \
		APP_NAME=$@
