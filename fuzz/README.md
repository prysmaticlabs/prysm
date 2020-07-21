# Prysm Fuzz Testing

[![fuzzit](https://app.fuzzit.dev/badge?org_id=prysmaticlabs-gh)](https://app.fuzzit.dev/orgs/prysmaticlabs-gh/dashboard)

## Adding a fuzz test

Fuzz testing attempts to find crash level bugs within the tested code paths, but could also be used
as a sanity check certain logic. 

### 1) Determining an ideal target

A fuzz test inputs pseudo-random data to a given method and attempts to find input data that tests
as many code branches as possible. When choosing a target to test, consider that the method under
test should be as stateless as possible. While stateful methods (i.e. methods that use a cache), 
can be tested, they are often hard to reproduce in a regression test. Consider disabling any caches
or persistence layers if possible. 

### 2) Writing a fuzz test

First, you need to determine in your input data. The current test suite uses SSZ encoded bytes to
deserialize to input objects. 

_Example: Block header input data_

```go
type InputBlockWithPrestate struct {
	StateID uint16
	Block   *ethpb.BeaconBlock
}
```

You'll also want to add that struct to `//fuzz:ssz_generated_files` to generate the custom fast SSZ
methods for serialization to improve test performance.

Your fuzz test must accept a single argument of type `[]byte`. The return types are ignored by 
libfuzzer, but might be useful for other applications such as 
[beacon-fuzz](https://github.com/sigp/beacon-fuzz). Be sure to name your test file with the
`_fuzz.go` suffix for consistency. 

```go
func MyExampleFuzz(b []byte) {
    input := &MyFuzzInputData{}
    if err := ssz.Unmarshal(b, input); err != nil {
       return // Input bytes doesn't serialize to input object.
    }
    
    result, err := somePackage.MethodUnderTest(input)
    if err != nil {
       // Input was invalid for processing, but the method didn't panic so that's OK.
       return 
    }
    // Optional: sanity check the resulting data.
    if result < 0 {
       panic("MethodUnderTest should never return a negative number") // Fail!
    }
}
```

### 3) Add your fuzz target to fuzz/BUILD.bazel

Since we are using some custom rules to generate the fuzz test instrumentation and appropriate
libfuzz testing suite, we cannot rely on gazelle to generate these targets for us.

```starlark
go_fuzz_test(
    name = "example_fuzz_test",
    srcs = [
        "example_fuzz.go",
    ] + COMMON_SRCS, # common and input type files.
    corpus = "example_corpus",
    corpus_path = "fuzz/example_corpus", # Path from root of project
    func = "MyExampleFuzz",
    importpath = IMPORT_PATH,
    deps = [
        # Deps used in your fuzz test.
    ] + COMMON_DEPS,
)
```

Be sure to add your target to the test suite at `//fuzz:fuzz_tests`.

### 4) Run your fuzz test

To run your fuzz test you must manually target it with bazel test and run with the config flag 
`--config=fuzz`.

```
bazel test //fuzz:example_fuzz_test --config=fuzz
```

## Running fuzzit regression tests

To run fuzzit regression tests, you can run the fuzz test suite with the 1--config=fuzzit`
configuration flag. Note: This requires docker installed on your machine. See 
[fuzzitdev/fuzzit#58](https://github.com/fuzzitdev/fuzzit/issues/58).

```
bazel test //fuzz:fuzz_tests --config=fuzzit
```

If the same command above is run with the FUZZIT_API_KEY environment variable set, then the fuzzit
test targets will be uploaded and restarted at https://app.fuzzit.dev.
