# dubplate version: v1.0.0-2-g61f3327-dirty (manually edited)

ROOT_DIR ?= $(shell git rev-parse --show-toplevel)
UNTRACKED ?= $(shell test -z "$(shell git ls-files --others --exclude-standard "$(ROOT_DIR)")" || echo -untracked)
VERSION ?= $(shell git describe --tags --dirty --always)$(UNTRACKED)

BUILD_DIR ?= ./bin

$(BUILD_DIR):
	mkdir -p $@

clean:
	rm $(BUILD_DIR)/*

version:
	@echo ${VERSION}

cmd-all: binary test-binary-version-output image
