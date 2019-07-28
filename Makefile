BINARIES ?= packing

include ./go.Makefile

packing:
	$(MAKE) cmd-all \
		APP_NAME=$@
