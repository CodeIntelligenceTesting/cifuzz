# How to write a fuzz test


## How to build/compile your fuzz tests

### CMake

If you created a fuzz test with `cifuzz create` and followed the 
instructions printed by the command, the fuzz test is now built as part 
of the regular CMake build. 
You can build the replayer binary (see [README](../README.md#regression-testing)
for more information about this) for your fuzz test by:

```
cmake -S . -B build
# for building the whole project incl. all fuzz tests
make -C build
# for building a single fuzz test
make -C build my_fuzz_test
```

Of course you can also build the fuzz test via the CMake integration 
of your IDE.

## How to convert/cast the fuzzer data into the data types you need

### C/C++

You might have to convert/cast the input parameters 
`const uint8_t *data, size_t size` to other types to call your 
functions. A useful tool for this is the [FuzzedDataProvider](https://github.com/google/fuzzing/blob/master/docs/split-inputs.md#fuzzed-data-provider).
If you use Clang/LLVM as your compiler of choice you can use it directly with 
`#include <fuzzer/FuzzedDataProvider.h>`, otherwise you can just copy 
the source file and add it to your project. 
An example can look like this:

``` cpp
#include <stdio.h>
#include <cifuzz/cifuzz.h>
#include <fuzzer/FuzzedDataProvider.h>

FUZZ_TEST_SETUP() {}

FUZZ_TEST(const uint8_t *data, size_t size) {

  FuzzedDataProvider fuzzed_data(data, size);
  int my_int = fuzzed_data.ConsumeIntegral<int8_t>();
  std::string my_string = fuzzed_data.ConsumeRandomLengthString();

  myFunction(my_int, my_string);
}
```

## Best Practices

### Regression Test / Replayer

When creating a new fuzz test we recommend to use the regression test
mode (see [README](../README.md#regression-testing)) to maintain a 
fast and responsive development cycle. 
When you run the CMake target from your IDE, the fuzz test is 
executed in regression test mode.
![fuzz test in CMake](/docs/assets/cmake_clion.gif)

You can also run the regression test through the replayer binary
by building and running the CMake target manually:

#### CMake
``` bash
cmake -S . -B build
make -C build my_fuzz_test
./build/my_fuzz_test
```
