FROM golang:1.15.5-buster AS builder
MAINTAINER libin <civet148@126.com>

ENV SRC_DIR /stos-wallet

RUN apt-get update && apt-get install -y ca-certificates llvm clang mesa-opencl-icd ocl-icd-opencl-dev jq hwloc libhwloc-dev make

ENV TINI_VERSION v0.18.0
RUN set -x \
  && cd /tmp \
  && wget -q -O tini https://github.com/krallin/tini/releases/download/$TINI_VERSION/tini \
  && chmod +x tini

RUN go env -w GOPROXY=https://goproxy.cn,https://goproxy.io,direct

RUN mkdir -p $SRC_DIR
WORKDIR $SRC_DIR

COPY . .
RUN git submodule update --init --recursive 
RUN make -C extern/filecoin-ffi
COPY go.mod .
COPY go.sum .
RUN go mod download && go mod graph | awk '{if ($1 !~ "@") print $2}' | xargs go get -v

RUN go build -o /stos-wallet

FROM ubuntu:20.04

COPY --from=builder /stos-wallet /usr/local/bin/stos-wallet
COPY --from=builder /tmp/tini /sbin/tini

ENV HOME_PATH /data

VOLUME $HOME_PATH

CMD ["/sbin/tini", "--"]
