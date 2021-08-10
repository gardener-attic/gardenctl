
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

DATE=$(shell date -u +%Y-%m-%d)
VERSION=$(shell cat VERSION | sed 's/-dev//g')
LDFLAGS=-w

.PHONY: build
build:
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build \
		-mod=vendor \
		-ldflags "${LDFLAGS} -X github.com/gardener/gardenctl/pkg/cmd.version=${VERSION} -X github.com/gardener/gardenctl/pkg/cmd.buildDate=${DATE}" \
		-o bin/linux-amd64/gardenctl-linux-amd64 cmd/gardenctl/main.go

	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 GO111MODULE=on go build \
		-mod=vendor \
		-ldflags "${LDFLAGS} -X github.com/gardener/gardenctl/pkg/cmd.version=${VERSION} -X github.com/gardener/gardenctl/pkg/cmd.buildDate=${DATE}" \
		-o bin/darwin-amd64/gardenctl-darwin-amd64 cmd/gardenctl/main.go

.PHONY: debug-build
debug-build: LDFLAGS=
debug-build: build

.PHONY: clean
clean:
	@rm -rf bin/

.PHONY: check
check:
	@.ci/check

.PHONY: test
test:
	@.ci/test

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod vendor
	@GO111MODULE=on go mod tidy
