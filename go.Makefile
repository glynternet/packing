# dubplate version: v1.0.0-2-g61f3327 (manually edited)

BUILD_DIR ?= ./bin
OUTBIN ?= $(BUILD_DIR)/$(APP_NAME)-$(VERSION)

VERSION_VAR ?= main.version
LDFLAGS = -ldflags "-w -X $(VERSION_VAR)=$(VERSION)"
GOBUILD_FLAGS ?= -installsuffix cgo -a $(LDFLAGS) -o $(OUTBIN)
GOBUILD_ENVVARS ?= CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH)
GOBUILD_CMD ?= $(GOBUILD_ENVVARS) go build $(GOBUILD_FLAGS)

OS ?= linux
ARCH ?= amd64

dummy:
	@echo No default rule set yet

clean:
	rm $(BUILD_DIR)/*

$(BUILD_DIR):
	mkdir -p $@

cmd-all: binary test-binary-version-output

binary: $(BUILD_DIR)
	$(GOBUILD_CMD) ./cmd/$(APP_NAME)

binaries: $(BINARIES)

$(BINARIES):
	$(MAKE) cmd-all \
		APP_NAME=$@

test-binary-version-output: VERSION_CMD ?= $(OUTBIN) version
test-binary-version-output:
	@echo testing output of $(VERSION_CMD)
	test "$(shell $(VERSION_CMD))" = "$(VERSION)" && echo PASSED
