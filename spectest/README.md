# Spec Tests

Spec testing vectors: https://github.com/ethereum/eth2.0-spec-tests

To run all `mainnet` spec tests:

```bash
bazel test //... --test_tag_filters=spectest
```

Minimal tests require `--define ssz=minimal` setting and are not triggered
automatically when `//...` is selected. One can run minimal tests manually, though:

```bash
bazel test //spectest/minimal/phase0/epoch_processing:go_default_test  --test_tag_filters=spectest --define ssz=minimal
bazel test //spectest/minimal/phase0/operations:go_default_test  --test_tag_filters=spectest --define ssz=minimal
bazel test //spectest/minimal/phase0/rewards:go_default_test  --test_tag_filters=spectest --define ssz=minimal
bazel test //spectest/minimal/phase0/sanity:go_default_test  --test_tag_filters=spectest --define ssz=minimal
bazel test //spectest/minimal/phase0/shuffling/core/shuffle:go_default_test  --test_tag_filters=spectest --define ssz=minimal
bazel test //spectest/minimal/phase0/ssz_static:go_default_test  --test_tag_filters=spectest --define ssz=minimal
```
