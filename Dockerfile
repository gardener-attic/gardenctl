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

# build gardenctl binary
FROM golang:1.13.0
RUN mkdir -p /go/src/github.com/gardener/gardenctl &&\
    cd /go/src/github.com/gardener &&\
    git clone https://github.com/gardener/gardenctl.git &&\
    cd ./gardenctl &&\
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo -o gardenctl cmd/gardenctl/main.go

# minimal Ubuntu LTS version
FROM ubuntu:18.04

COPY --from=0 /go/src/github.com/gardener/gardenctl/gardenctl .
COPY clusters /root/clusters
COPY config /root/.garden/config

# run as root in root
USER root
WORKDIR /

# install basic tools
RUN apt-get --yes update;\
    apt-get --yes install curl;\
    apt-get --yes install tree;\
    apt-get --yes install silversearcher-ag;\
    apt-get --yes install htop;\
    apt-get --yes install less;\
    apt-get --yes install vim;\
    apt-get --yes install tmux;\
    apt-get --yes install bash-completion;\
    curl -sL https://github.com/jingweno/ccat/releases/download/v1.1.0/linux-amd64-1.1.0.tar.gz -o ccat.tar.gz && tar -zxvf ccat.tar.gz linux-amd64-1.1.0/ccat && mv linux-amd64-1.1.0/ccat /bin/cat && rm -rf linux-amd64-1.1.0 ccat.tar.gz && chmod 755 /bin/cat;\
    curl -sL http://stedolan.github.io/jq/download/linux64/jq -o /bin/jq && chmod 755 /bin/jq;\
    curl -sL https://github.com/bronze1man/yaml2json/raw/master/builds/linux_amd64/yaml2json -o /bin/yaml2json && chmod 755 /bin/yaml2json;\
    # remove package lists to safe space
    rm -rf /var/lib/apt/lists

# install network tools
RUN apt-get --yes update;\
    apt-get --yes install dnsutils;\
    apt-get --yes install netcat-openbsd;\
    apt-get --yes install iproute2;\
    apt-get --yes install dstat;\
    apt-get --yes install ngrep;\
    apt-get --yes install tcpdump;\
    # remove package lists to safe space
    rm -rf /var/lib/apt/lists

# install Kubernetes CLI
RUN curl -sL https://storage.googleapis.com/kubernetes-release/release/$(curl -sL https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl && chmod 755 /usr/local/bin/kubectl

# install minimal python
RUN apt-get --yes install python-minimal;\
    curl -sL https://bootstrap.pypa.io/get-pip.py -o get-pip.py;\
    python get-pip.py;\
    rm get-pip.py

# launch bash
ENTRYPOINT ["/bin/bash"]

# install AWS CLI
RUN pip install awscli

# install Azure CLI
RUN apt-get --yes update;\
    apt-get --yes install lsb-release gnupg apt-transport-https;\
    AZ_REPO="$(lsb_release -cs)";\
    echo "deb https://packages.microsoft.com/repos/azure-cli $AZ_REPO main" | tee /etc/apt/sources.list.d/azure-cli.list;\
    curl -sL https://packages.microsoft.com/keys/microsoft.asc | apt-key add -;\
    apt-get --yes update && apt-get --yes install azure-cli;\
    apt-get --yes --purge remove lsb-release gnupg apt-transport-https;\
    # remove package lists to safe space
    rm -rf /var/lib/apt/lists

# install GCP CLI
RUN apt-get --yes update;\
    apt-get --yes install lsb-release gnupg apt-transport-https;\
    GCP_REPO="cloud-sdk-$(lsb_release -cs)";\
    echo "deb http://packages.cloud.google.com/apt $GCP_REPO main" | tee /etc/apt/sources.list.d/google-cloud-sdk.list;\
    curl -sL https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -;\
    apt-get --yes update && apt-get --yes install google-cloud-sdk;\
    apt-get --yes --purge remove lsb-release gnupg apt-transport-https;\
    # remove package lists to safe space
    rm -rf /var/lib/apt/lists

# install OpenStack CLI
RUN pip install python-novaclient python-glanceclient python-cinderclient python-swiftclient
# install Gardener CLI
RUN mkdir -p /opt/gardenctl/bin &&\
    mv gardenctl /opt/gardenctl/bin/gardenctl &&\
    ln -s /opt/gardenctl/bin/gardenctl /usr/local/bin/gardenctl &&\
    gardenctl completion bash > /root/gardenctl_bash_completion.sh &&\
    echo ". /etc/profile" >> /root/.bashrc &&\
    echo ". /root/gardenctl_bash_completion.sh" >> /root/.bashrc
