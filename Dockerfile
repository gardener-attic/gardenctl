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

FROM golang:1.10

ENV PATH $PATH:/root/google-cloud-sdk/bin

RUN apt-get update &&\
    apt-get upgrade -qy &&\
    apt-get install -qy git &&\
    apt-get install -qy jq &&\
    apt-get install -qy python &&\
    apt-get install -qy sudo python python-pip python-setuptools &&\
    pip install awscli &&\
    pip install azure-cli &&\
    apt-get update && apt-get install -y apt-transport-https &&\
    curl -sSL https://sdk.cloud.google.com | bash

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl &&\
    chmod +x ./kubectl &&\
    sudo mv ./kubectl /usr/local/bin/kubectl

RUN mkdir -p /go/src/github.com/gardener/gardenctl &&\
    cd /go/src/github.com/gardener &&\
    git clone https://github.com/gardener/gardenctl.git &&\
    go install github.com/gardener/gardenctl
