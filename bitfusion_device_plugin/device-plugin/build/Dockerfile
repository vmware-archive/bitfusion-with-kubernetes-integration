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
WORKDIR /gopath/src/device-plugin

COPY ./device-plugin .
RUN go build -o bitfusion-device-plugin ./cmd/
RUN cp bitfusion-device-plugin /usr/bin/bitfusion-device-plugin \
&& chmod +x ./cmd/start.sh \
&& cp ./cmd/*.sh /usr/bin/


FROM ubuntu:18.04

WORKDIR /
COPY --from=builder /usr/bin/bitfusion-device-plugin /
COPY --from=builder /usr/bin/start.sh /

CMD ["./start.sh"]
