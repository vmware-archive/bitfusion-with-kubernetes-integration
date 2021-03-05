# Copyright 2020 VMware, Inc.
#
# SPDX-License-Identifier: Apache-2.0

FROM ubuntu:18.04 as builder


RUN apt update && apt install curl -y
RUN \
  mkdir -p /goroot && \
  curl https://storage.googleapis.com/golang/go1.14.linux-amd64.tar.gz | tar xvzf - -C /goroot --strip-components=1
# Set environment variables.
ENV GOROOT /goroot
ENV GOPATH /gopath
ENV PATH $GOROOT/bin:$GOPATH/bin:$PATH

# Define working directory.
WORKDIR /gopath/src/webhook

COPY ./webhook .

RUN go build -o sidecar-injector ./cmd/
RUN cp sidecar-injector /usr/bin/sidecar-injector

FROM ubuntu:18.04

WORKDIR /


COPY --from=builder /usr/bin/sidecar-injector /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/sidecar-injector"]