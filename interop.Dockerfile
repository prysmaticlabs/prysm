FROM debian:9.9-slim AS builder

ENV BAZEL_VERSION 0.29.0

# Creating the man pages directory to deal with the slim variants not having it.
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl gnupg git\
 && echo "deb [arch=amd64] http://storage.googleapis.com/bazel-apt stable jdk1.8" > \
         /etc/apt/sources.list.d/bazel.list \
 && curl https://bazel.build/bazel-release.pub.gpg | apt-key add - \
 && apt-get update && apt-get install -y --no-install-recommends bazel=${BAZEL_VERSION} \
 && apt-get purge --auto-remove -y curl gnupg \
 && rm -rf /etc/apt/sources.list.d/bazel.list \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
COPY . /workspace/.

RUN bazel version

# Build binaries for minimal configuration.
RUN bazel build --define ssz=minimal \
  //beacon-chain \
  //validator \
  //tools/interop/convert-keys


FROM alpine:3

COPY --from=builder /workspace/bazel-bin/beacon-chain/linux_amd64_stripped/beacon-chain .
COPY --from=builder /workspace/bazel-bin/validator/linux_amd64_stripped/validator .
COPY --from=builder /workspace/bazel-bin/tools/interop/convert-keys/linux_amd64_stripped/convert-keys .

COPY scripts/interop_start.sh start.sh

ENTRYPOINT ["start.sh"]
