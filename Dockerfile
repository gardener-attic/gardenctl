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

COPY clusters /root/clusters
COPY config /root/.garden/config

RUN apt-get update &&\
    apt-get upgrade -qy &&\
    apt-get install -qy git &&\
    apt-get install -qy jq &&\
    apt-get install -qy python python-pip python3-pip python-setuptools &&\
    pip install awscli &&\
    pip install azure-cli &&\
    pip uninstall pyopenssl -y &&\
    pip install pyopenssl &&\
    pip3 install python-openstackclient &&\
    apt-get update && apt-get install -y apt-transport-https &&\
    curl -sSL https://sdk.cloud.google.com | bash &&\
    ln -s /root/google-cloud-sdk/bin/gcloud /usr/bin/gcloud &&\
    curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl &&\
    chmod +x ./kubectl &&\
    mv ./kubectl /usr/local/bin/kubectl &&\
    mkdir -p /go/src/github.com/gardener/gardenctl &&\
    cd /go/src/github.com/gardener &&\
    git clone https://github.com/gardener/gardenctl.git &&\
    go install github.com/gardener/gardenctl &&\
    /bin/bash -c "ln -s /go/bin/gardenctl /usr/local/bin/gardenctl" &&\
    apt-get install bash-completion &&\
    gardenctl completion; mv gardenctl_completion.sh /root/gardenctl_completion.sh &&\
    echo ". /etc/profile" >> /root/.bashrc &&\
    echo ". /root/gardenctl_completion.sh" >> /root/.bashrc
