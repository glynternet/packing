OUT_FILE ?= ../cmd/packing-server/elm.js

ELM_SOURCES ?= ./src/Main.elm

default: elm-js

elm-js:
	elm make $(ELM_SOURCES) --output=$(OUT_FILE)
	#elm make --optimize $(ELM_SOURCES) --output=$(OUT_FILE)

