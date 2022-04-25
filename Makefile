#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
default: help

VERSION ?= latest
RELEASE_SRC = apisix-go-plugin-runner-${VERSION}-src

GITSHA ?= $(shell git rev-parse --short=7 HEAD 2> /dev/null || echo '')
OSNAME ?= $(shell uname -s | tr A-Z a-z)
OSARCH ?= $(shell uname -m | tr A-Z a-z)
PWD ?= $(shell pwd)
ifeq ($(OSARCH), x86_64)
	OSARCH = amd64
endif

VERSYM=main._buildVersion
GITSHASYM=main._buildGitRevision
BUILDOSSYM=main._buildOS
GO_LDFLAGS ?= "-X '$(VERSYM)=$(VERSION)' -X '$(GITSHASYM)=$(GITSHA)' -X '$(BUILDOSSYM)=$(OSNAME)/$(OSARCH)'"

.PHONY: build
build:
	cd cmd/go-runner && \
	go build $(GO_BUILD_FLAGS) -ldflags $(GO_LDFLAGS) && \
	mv go-runner ../..

.PHONY: lint
lint:
	golangci-lint run --verbose ./...

.PHONY: test
test:
	go test -race -cover -coverprofile=coverage.txt ./...

.PHONY: release-src
release-src: compress-tar
	gpg --batch --yes --armor --detach-sig $(RELEASE_SRC).tgz
	shasum -a 512 $(RELEASE_SRC).tgz > $(RELEASE_SRC).tgz.sha512

	mkdir -p release
	mv $(RELEASE_SRC).tgz release/$(RELEASE_SRC).tgz
	mv $(RELEASE_SRC).tgz.asc release/$(RELEASE_SRC).tgz.asc
	mv $(RELEASE_SRC).tgz.sha512 release/$(RELEASE_SRC).tgz.sha512

.PHONY: compress-tar
compress-tar:
	tar -zcvf $(RELEASE_SRC).tgz \
	./cmd \
	./internal \
	./pkg \
	LICENSE \
	Makefile \
	NOTICE \
	go.mod \
	go.sum \
	*.md

.PHONY: help
help:
	@echo Makefile rules:
	@echo
	@grep -E '^### [-A-Za-z0-9_]+:' Makefile | sed 's/###/   /'
