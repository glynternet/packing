# dubplate version: v1.0.0-2-g61f3327-dirty (manually edited)

dockerfile:
	$(MAKE) Dockerfile.$(APP_NAME)

# The change below needs improving and upstreaming

Dockerfile.%:
	sed 's/{{APP_NAME}}/$(subst Dockerfile.,,$@)/g' Dockerfile.template | sed 's/{{BIN_NAME}}/$(BIN_NAME)/g' > $(BUILD_DIR)/$@

image: Dockerfile.$(APP_NAME) check-docker-username
	docker build \
		--tag $(DOCKER_USERNAME)/$(APP_NAME):$(VERSION) \
		-f $(BUILD_DIR)/Dockerfile.$(APP_NAME) \
		./bin

check-docker-username:
ifndef DOCKER_USERNAME
	$(error DOCKER_USERNAME var not defined)
endif
