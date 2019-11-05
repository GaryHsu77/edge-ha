FROM golang:1.12-stretch

ARG ARCH

ENV ARCH=${ARCH} \
    GO111MODULE=on

RUN dpkg --add-architecture ${ARCH} && \
    sed -i "s/deb.debian.org/cdn-fastly.deb.debian.org/" /etc/apt/sources.list && \
    apt update && apt install -y --no-install-recommends \
        mosquitto

COPY vendor vendor
COPY cmd cmd
COPY go.mod ./
COPY go.sum ./
COPY Makefile ./
COPY cmd/cloud/run.sh /usr/sbin/
RUN make mqtt-exporter && mv ./build/${ARCH}/mqtt-exporter /usr/sbin/
CMD "/usr/sbin/run.sh"