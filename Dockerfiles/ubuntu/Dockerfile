# Copyright 2020 VMware, Inc.
#
# SPDX-License-Identifier: Apache-2.0

FROM nvcr.io/nvidia/cuda:11.2.1-cudnn8-devel-ubuntu18.04

RUN apt-get update && apt-get install -y --no-install-recommends \
        apt-utils \
        build-essential \
        ca-certificates \
        curl \
        kmod

# Start of bitfusion
RUN apt-get install -y --no-install-recommends \
        wget uuid libjsoncpp1 librdmacm1 libssl-dev libibverbs1 libnuma1 libcapstone3 libnl-3-200 libnl-route-3-200 open-vm-tools && \
    rm -rf /var/lib/apt/lists/*

# TODO: Make this work with Bitfusion GA version
RUN cd /tmp && \
    curl -fSslL -O https://packages.vmware.com/bitfusion/ubuntu/18.04/bitfusion-client-ubuntu1804_3.0.0-11_amd64.deb && \
    apt-get install -y ./bitfusion-client-ubuntu1804_3.0.0-11_amd64.deb && \
    rm -rf /tmp/*
