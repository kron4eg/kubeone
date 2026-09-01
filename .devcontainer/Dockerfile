# Copyright 2026 The KubeOne Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.27

# Editor/IDE tooling.
ARG GOPLS_VERSION=latest
ARG DLVS_VERSION=latest

# Non-root user for the devcontainer.
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=1000

# Base utilities needed by the Makefile and scripts.
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        bash \
        fish \
        ca-certificates \
        curl \
        git \
        jq \
        make \
        openssh-client \
        sudo \
        unzip

# Create the non-root user and give it ownership of the Go workspace.
RUN groupadd --gid "${USER_GID}" "${USERNAME}" \
    && useradd --uid "${USER_UID}" --gid "${USER_GID}" --create-home --shell /bin/bash "${USERNAME}" \
    && echo "${USERNAME} ALL=(root) NOPASSWD:ALL" > "/etc/sudoers.d/${USERNAME}" \
    && chmod 0440 "/etc/sudoers.d/${USERNAME}" \
    && mkdir -p /go \
    && chown -R "${USERNAME}:${USERNAME}" /go

# Go tooling installed into the `vscode` user's GOPATH (/go/bin), which is on PATH.
USER "${USERNAME}"
RUN    go install "golang.org/x/tools/gopls@${GOPLS_VERSION}" \
    && go install "github.com/go-delve/delve/cmd/dlv@${DLVS_VERSION}"

USER root
