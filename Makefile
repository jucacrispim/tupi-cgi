GOCMD=go
GOBUILD=$(GOCMD) build -trimpath
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test -v ./... -trimpath
GOFUNCTEST=$(GOCMD) test ./functionaltests -v
BUILD_DIR=build
PLUGIN_BIN=./$(BUILD_DIR)/cgi_plugin.so

BIN_PATH=./$(BUILD_DIR)/$(BIN_NAME)
OUTFLAG=-o $(BIN_PATH)
PLUGIN_MODE_FLAG=-buildmode=plugin
PLUGIN_FILE=plugin.go

SCRIPTS_DIR=./scripts/

TESTDATA_DIR=./testdata
CGI_FILE=$(TESTDATA_DIR)/something.go
CGI_BIN=$(TESTDATA_DIR)/something


.PHONY: build # - Creates the binary under the build/ directory
build: buildplugin


.PHONY: buildplugin # - Creates the plugin .so binary under the build/ directory
buildplugin:
	$(GOBUILD) -o $(PLUGIN_BIN) $(PLUGIN_MODE_FLAG) $(PLUGIN_FILE)

.PHONY: buildcgi # - Builds the cgi bin for tests
buildcgi:
	$(GOBUILD) -o $(CGI_BIN) $(CGI_FILE)

.PHONY: test # - Run all tests
test: buildcgi
	$(GOTEST)

.PHONY: functest # - Run functional tests
functest: build
	$(GOFUNCTEST)

.PHONY: setupenv # - Install needed tools for tests/docs
setupenv:
	$(SCRIPTS_DIR)/env.sh setup-env

.PHONY: docs # - Build documentation
docs:
	$(SCRIPTS_DIR)/env.sh build-docs

.PHONY: coverage # - Run tests and check coverage
coverage: cov

cov: buildcgi
	$(SCRIPTS_DIR)/check_coverage.sh

.PHONY: run # - Run the program. You can use `make run ARGS="-host :9090 -root=/"`
run:
	$(GOBUILD) $(OUTFLAG)
	$(BIN_PATH) $(ARGS)

.PHONY: clean # - Remove the files created during build
clean:
	rm -rf $(BUILD_DIR)

.PHONY: install # - Copy the binary to the path
install: build
	go install

.PHONY: uninstall # - Remove the binary from path
uninstall:
	go clean -i github.com/jucacrispim/tupi-cgi


all: build test install

.PHONY: help  # - Show this help text
help:
	@grep '^.PHONY: .* #' Makefile | sed 's/\.PHONY: \(.*\) # \(.*\)/\1 \2/' | expand -t20
