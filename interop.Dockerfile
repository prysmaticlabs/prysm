FROM gcr.io/prysmaticlabs/build-agent AS builder

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
