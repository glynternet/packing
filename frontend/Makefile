OUT_FILE ?= ../cmd/packing-server/elm.js

ELM_SOURCES ?= ./src/Main.elm ./src/HostedMain.elm
ELM ?= ./node_modules/.bin/elm

default: elm-js

elm-js:
	$(ELM) make $(ELM_SOURCES) --output=$(OUT_FILE)

.PHONY: default elm-js setup test test-browser check-format format
setup:
	npm ci
	npm exec -- playwright install chromium

test:
	npm test

test-browser:
	npm run test:browser

check-format:
	npm exec -- elm-format src tests --validate

format:
	npm exec -- elm-format src tests --yes
