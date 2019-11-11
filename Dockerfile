# =================================================================================
#  Debug docker env
# =================================================================================
FROM golang:1.12-stretch

ARG ARCH

ENV ARCH=${ARCH} \
    GO111MODULE=on

RUN dpkg --add-architecture ${ARCH} && \
    sed -i "s/deb.debian.org/cdn-fastly.deb.debian.org/" /etc/apt/sources.list && \
    apt update && apt install -y --no-install-recommends crossbuild-essential-armhf \
        mosquitto