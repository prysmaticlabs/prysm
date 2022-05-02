# Build Stage
FROM --platform=linux/amd64 ubuntu:20.04 as builder
RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y git wget python

RUN wget https://go.dev/dl/go1.18.1.linux-amd64.tar.gz
RUN tar -C /usr/local -xvf go1.18.1.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin

RUN apt install -y apt-transport-https curl gnupg 
RUN curl -fsSL https://bazel.build/bazel-release.pub.gpg | gpg --dearmor > bazel.gpg
RUN mv bazel.gpg /etc/apt/trusted.gpg.d/
RUN echo "deb [arch=amd64] https://storage.googleapis.com/bazel-apt stable jdk1.8" | tee /etc/apt/sources.list.d/bazel.list
RUN apt update && apt install -y bazel-5.0.0

ADD . /prysm
WORKDIR /prysm

RUN bazel-5.0.0 build //cmd/validator:validator
ADD . /mayhem-prysm

FROM --platform=linux/amd64 ubuntu:20.04

COPY --from=builder /prysm/bazel-bin/cmd/validator/validator_/validator /

