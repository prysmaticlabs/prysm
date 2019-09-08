FROM gcr.io/prysmaticlabs/build-agent AS builder

WORKDIR /workspace

# Copy WORKSPACE and install golang. WORKSPACE doesn't change very often.
COPY WORKSPACE /workspace/.
COPY BUILD.bazel /workspace/.
RUN bazel build --jobs=auto --remote_cache= @io_bazel_rules_go//:go_info

# Copy everything else.
COPY . /workspace/.

# Build binaries for minimal configuration.
RUN bazel build --define ssz=minimal --jobs=auto --remote_cache= \
  //beacon-chain \
  //validator \
  //tools/interop/convert-keys


FROM alpine:3

COPY --from=builder /workspace/bazel-bin/beacon-chain/linux_amd64_stripped/beacon-chain .
COPY --from=builder /workspace/bazel-bin/validator/linux_amd64_stripped/validator .
COPY --from=builder /workspace/bazel-bin/tools/interop/convert-keys/linux_amd64_stripped/convert-keys .

RUN mkdir /launch

COPY scripts/interop_start.sh /launch/start.sh

ENTRYPOINT ["start.sh"]
