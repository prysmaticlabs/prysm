bazel build //fuzz:block_fuzz_test_libfuzzer_bundle //fuzz:state_fuzz_test_libfuzzer_bundle --config=fuzz 
gsutil cp bazel-bin/fuzz/block_fuzz_test_libfuzzer_bundle.zip gs://builds.prysmaticlabs.appspot.com/libfuzzer_asan_blocks/fuzzer-build-$(date +%Y%m%d%H%M).zip
gsutil cp bazel-bin/fuzz/state_fuzz_test_libfuzzer_bundle.zip gs://builds.prysmaticlabs.appspot.com/libfuzzer_asan_state/fuzzer-build-$(date +%Y%m%d%H%M).zip
gsutil cp bazel-bin/fuzz/ssz_encoder_attestations_test_libfuzzer_bundle.zip gs://builds.prysmaticlabs.appspot.com/libfuzzer_asan_ssz_encoder_attestations/fuzzer-build-$(date +%Y%m%d%H%M).zip
