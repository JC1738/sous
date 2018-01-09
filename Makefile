SHELL := /usr/bin/env bash

XDG_DATA_HOME ?= $(HOME)/.local/share
DEV_POSTGRES_DIR ?= $(XDG_DATA_HOME)/sous/postgres
DEV_POSTGRES_DATA_DIR ?= $(DEV_POSTGRES_DIR)/data
PGPORT ?= 6543

DB_NAME = sous
TEST_DB_NAME = sous_test_template

LIQUIBASE_DEFAULTS := ./dev_support/liquibase/liquibase.properties
LIQUIBASE_SERVER := jdbc:postgresql://localhost:$(PGPORT)
LIQUIBASE_SHARED_FLAGS = --changeLogFile=database/changelog.xml --defaultsFile=./dev_support/liquibase/liquibase.properties

LIQUIBASE_FLAGS := --url $(LIQUIBASE_SERVER)/$(DB_NAME) $(LIQUIBASE_SHARED_FLAGS)
LIQUIBASE_TEST_FLAGS := --url $(LIQUIBASE_SERVER)/$(TEST_DB_NAME) $(LIQUIBASE_SHARED_FLAGS)

SQLITE_URL := https://sqlite.org/2017/sqlite-autoconf-3160200.tar.gz
GO_VERSION := 1.9.2
DESCRIPTION := "Sous is a tool for building, testing, and deploying applications, using Docker, Mesos, and Singularity."
URL := https://github.com/opentable/sous

TAG_TEST := git describe --exact-match --abbrev=0 2>/dev/null
ifeq ($(shell $(TAG_TEST) ; echo $$?), 128)
GIT_TAG := 0.0.0
else
GIT_TAG := $(shell $(TAG_TEST))
endif

# install-dev uses DESC and DATE to make a git described, timestamped dev build.
DESC := $(shell git describe)
DATE := $(shell date +%Y-%m-%dT%H-%M-%S)
DEV_VERSION := "$(DESC)-devbuild-$(DATE)"

# Sous releases are tagged with format v0.0.0. semv library
# does not understand the v prefix, so this lops it off.
SOUS_VERSION := $(shell echo $(GIT_TAG) | sed 's/^v//')

ifeq ($(shell git diff-index --quiet HEAD ; echo $$?),0)
COMMIT := $(shell git rev-parse HEAD)
else
COMMIT := DIRTY
endif

ifndef SOUS_QA_DESC
QA_DESC := `pwd`/qa_desc.json
else
QA_DESC := $(SOUS_QA_DESC)
endif

FLAGS := "-X 'main.Revision=$(COMMIT)' -X 'main.VersionString=$(SOUS_VERSION)'"
BIN_DIR := artifacts/bin
DARWIN_RELEASE_DIR := sous-darwin-amd64_$(SOUS_VERSION)
LINUX_RELEASE_DIR := sous-linux-amd64_$(SOUS_VERSION)
RELEASE_DIRS := $(DARWIN_RELEASE_DIR) $(LINUX_RELEASE_DIR)
DARWIN_TARBALL := $(DARWIN_RELEASE_DIR).tar.gz
LINUX_TARBALL := $(LINUX_RELEASE_DIR).tar.gz
CONCAT_XGO_ARGS := -go $(GO_VERSION) -branch master -deps $(SQLITE_URL) --dest $(BIN_DIR) --ldflags $(FLAGS)
COVER_DIR := /tmp/sous-cover
TEST_VERBOSE := $(if $(VERBOSE),-v,)
SOUS_PACKAGES:= $(shell go list -f '{{.ImportPath}}' ./... | grep -v 'vendor')
SOUS_PACKAGES_WITH_TESTS:= $(shell go list -f '{{if len .TestGoFiles}}{{.ImportPath}}{{end}}' ./...)
SOUS_CONTAINER_IMAGES:= "docker images | egrep '127.0.0.1:5000|testregistry_'"

help:
	@echo --- options:
	@echo make clean
	@echo "make clean-containers: Destroy and delete local testing containers."
	@echo make coverage
	@echo make legendary
	@echo "make release:  Both linux and darwin"
	@echo "make setup-containers: pull and start containers for integration testing."
	@echo "make test: all tests"
	@echo "make test-unit: unit tests"
	@echo "make test-gofmt: gofmt tests"
	@echo "make test-integration: integration tests"
	@echo "make test-staticcheck: runs static code analysis against project packages."
	@echo "make wip: puts a marker file into workspace to prevent Travis from passing the build."
	@echo
	@echo "Add VERBOSE=1 for tons of extra output."

clean:
	rm -rf $(COVER_DIR)
	git ls-files -z -o --exclude=.cleanprotect --exclude-per-directory=.cleanprotect | xargs -0 rm -rf
	rm -f $(QA_DESC)

clean-containers: clean-container-certs clean-running-containers clean-container-images

clean-container-images:
	@if (( $$("$(SOUS_CONTAINER_IMAGES)" | wc -l) > 0 )); then echo 'found docker images, deleting:'; "$(SOUS_CONTAINER_IMAGES)" | awk '{ print $$3 }' | xargs docker rmi -f; fi

clean-container-certs:
	rm -f ./integration/test-registry/docker-registry/testing.crt

clean-running-containers:
	@if (( $$(docker ps -q | wc -l) > 0 )); then echo 'found running containers, killing:'; docker ps -q | xargs docker kill; fi
	@if (( $$(docker ps -aq | wc -l) > 0 )); then echo 'found container instances, deleting:'; docker ps -aq | xargs docker rm; fi

gitlog:
	git log `git describe --abbrev=0`..HEAD

install-dev:
	brew uninstall opentable/public/sous || true
	rm "$$(which sous)" || true
	go install -ldflags "-X main.VersionString=$(DEV_VERSION)"
	echo "Now run 'hash -r && sous version' to make sure you are using the dev version of sous."

install-brew:
	rm "$$(which sous)" || true
	brew uninstall opentable/public/sous || true
	brew install opentable/public/sous
	echo "Now run 'hash -r && sous version' to make sure you are using the homebrew-installed sous."

install-fpm:
	gem install --no-ri --no-rdoc fpm

install-jfrog:
	go get github.com/jfrogdev/jfrog-cli-go/jfrog

install-ggen:
	cd bin/ggen && go install ./

install-stringer:
	go get golang.org/x/tools/cmd/stringer

install-xgo:
	go get github.com/karalabe/xgo

install-govendor:
	go get github.com/kardianos/govendor

install-engulf:
	go get github.com/nyarly/engulf

install-staticcheck:
	go install -v ./vendor/honnef.co/go/tools/cmd/staticcheck

install-metalinter:
	go get github.com/alecthomas/gometalinter

install-linters: install-metalinter
	gometalinter --install > /dev/null

install-build-tools: install-xgo install-govendor install-engulf install-staticcheck

release: artifacts/$(DARWIN_TARBALL) artifacts/$(LINUX_TARBALL) artifacts/sous_$(SOUS_VERSION)_amd64.deb

artifactory: artifacts/sous_$(SOUS_VERSION)_amd64.deb
	jfrog rt upload -deb trusty/main/amd64 artifacts/sous_$(SOUS_VERSION)_amd64.deb opentable-ppa/pool/sous_$(SOUS_VERSION)_amd64.deb

deb-build: artifacts/sous_$(SOUS_VERSION)_amd64.deb

linux-build: artifacts/$(LINUX_RELEASE_DIR)/sous
	ln -sf ../$< dev_support/sous_linux

semvertagchk:
	@echo "$(SOUS_VERSION)" | egrep ^[0-9]+\.[0-9]+\.[0-9]+

sous-qa-setup: ./dev_support/sous_qa_setup/*.go ./util/test_with_docker/*.go
	go build $(EXTRA_GO_FLAGS) ./dev_support/sous_qa_setup


reject-wip:
	test ! -f workinprogress

wip:
	touch workinprogress
	git add workinprogress
	git commit --squash=HEAD -m "Making WIP" --no-gpg-sign --no-verify

coverage: $(COVER_DIR)
	engulf -s --coverdir=$(COVER_DIR) \
		--exclude '/vendor,integration/?,/bin/?,/dev_support/?,/util/test_with_docker/?,/examples/?,/util/cmdr/cmdr-example/?'\
		--exclude-files='raw_client.go$$,_generated.go$$'\
		--merge-base=_merged.txt ./...

legendary: coverage
	legendary --hitlist .cadre/coverage.vim /tmp/sous-cover/*_merged.txt

test: test-gofmt test-staticcheck test-unit test-integration

test-dev: test-staticcheck test-unit

test-staticcheck: install-staticcheck
	staticcheck -ignore "$$(cat staticcheck.ignore)" $(SOUS_PACKAGES)
	staticcheck -tags integration -ignore "$$(cat staticcheck.ignore)" github.com/opentable/sous/integration

test-metalinter: install-linters
	gometalinter --config gometalinter.json ./...

test-gofmt:
	bin/check-gofmt

test-unit:
	go test $(EXTRA_GO_FLAGS) $(TEST_VERBOSE) -timeout 3m -race $(SOUS_PACKAGES_WITH_TESTS)

test-integration: setup-containers
	SOUS_QA_DESC=$(QA_DESC) go test -timeout 30m $(EXTRA_GO_FLAGS)  $(TEST_VERBOSE) ./integration --tags=integration
	@date

$(QA_DESC): sous-qa-setup
	./sous_qa_setup --compose-dir ./integration/test-registry/ --out-path=$(QA_DESC)

setup-containers:  $(QA_DESC)

test-cli: setup-containers linux-build
	rm -rf integration/raw_shell_output/0*
	@date
	SOUS_QA_DESC=$(QA_DESC) go test $(EXTRA_GO_FLAGS) $(TEST_VERBOSE) -timeout 20m ./integration --tags=commandline


$(COVER_DIR):
	mkdir -p $@

artifacts/$(DARWIN_RELEASE_DIR)/sous:
	mkdir -p artifacts/$(DARWIN_RELEASE_DIR)
	cp -R doc/ artifacts/$(DARWIN_RELEASE_DIR)/doc
	cp README.md artifacts/$(DARWIN_RELEASE_DIR)
	cp LICENSE artifacts/$(DARWIN_RELEASE_DIR)
	mkdir -p $(BIN_DIR)
	xgo $(CONCAT_XGO_ARGS) --targets=darwin/amd64  ./
	mv $(BIN_DIR)/sous-darwin-10.6-amd64 $@

artifacts/$(LINUX_RELEASE_DIR)/sous:
	mkdir -p artifacts/$(LINUX_RELEASE_DIR)
	cp -R doc/ artifacts/$(LINUX_RELEASE_DIR)/doc
	cp README.md artifacts/$(LINUX_RELEASE_DIR)
	cp LICENSE artifacts/$(LINUX_RELEASE_DIR)
	mkdir -p $(BIN_DIR)
	xgo $(CONCAT_XGO_ARGS) --targets=linux/amd64  ./
	mv $(BIN_DIR)/sous-linux-amd64 $@

artifacts/$(LINUX_TARBALL): artifacts/$(LINUX_RELEASE_DIR)/sous
	cd artifacts && tar czv $(LINUX_RELEASE_DIR) > $(LINUX_TARBALL)

artifacts/$(DARWIN_TARBALL): artifacts/$(DARWIN_RELEASE_DIR)/sous
	cd artifacts && tar czv $(DARWIN_RELEASE_DIR) > $(DARWIN_TARBALL)

artifacts/sous_$(SOUS_VERSION)_amd64.deb: artifacts/$(LINUX_RELEASE_DIR)/sous
	fpm -s dir -t deb -n sous -v $(SOUS_VERSION) --description $(DESCRIPTION) --url $(URL) artifacts/$(LINUX_RELEASE_DIR)/sous=/usr/bin/sous
	mv sous_$(SOUS_VERSION)_amd64.deb artifacts/

$(DEV_POSTGRES_DATA_DIR):
	install -d -m 0700 $@
	initdb $@

$(DEV_POSTGRES_DATA_DIR)/postgresql.conf: $(DEV_POSTGRES_DATA_DIR) dev_support/postgres/postgresql.conf
	cp dev_support/postgres/postgresql.conf $@

postgres-start: $(DEV_POSTGRES_DATA_DIR)/postgresql.conf
	if ! (pg_isready -h localhost -p $(PGPORT)); then \
		postgres -D $(DEV_POSTGRES_DATA_DIR) -p $(PGPORT) & \
		until pg_isready -h localhost -p $(PGPORT); do sleep 1; done \
	fi
	createdb -h localhost -p $(PGPORT) $(DB_NAME) > /dev/null 2>&1 || true
	liquibase $(LIQUIBASE_FLAGS) update

postgres-test-prepare: $(DEV_POSTGRES_DATA_DIR)/postgresql.conf postgres-create-testdb

postgres-create-testdb:
	createdb -h localhost -p $(PGPORT) $(TEST_DB_NAME) > /dev/null 2>&1 || true
	liquibase $(LIQUIBASE_TEST_FLAGS) update

postgres-stop:
	pg_ctl stop -D $(DEV_POSTGRES_DATA_DIR) || true

postgres-connect:
	psql -h localhost -p $(PGPORT) sous

postgres-update-schema: postgres-start
	liquibase $(LIQUIBASE_FLAGS) update

postgres-clean: postgres-stop
	rm -r "$(DEV_POSTGRES_DIR)"

.PHONY: artifactory clean clean-containers clean-container-certs \
	clean-running-containers clean-container-images coverage deb-build \
	install-fpm install-jfrog install-ggen install-build-tools legendary release \
	semvertagchk test test-gofmt test-integration setup-containers test-unit \
	reject-wip wip staticcheck postgres-start postgres-stop postgres-connect \
	postgres-clean postgres-create-testdb

#liquibase --url jdbc:postgresql://127.0.0.1:6543/sous --changeLogFile=database/changelog.xml update
