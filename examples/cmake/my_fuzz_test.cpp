#include "src/explore_me.h"
#include <cifuzz/cifuzz.h>
#include <fuzzer/FuzzedDataProvider.h>
#include <stdio.h>

FUZZ_TEST_SETUP() {}

FUZZ_TEST(const uint8_t *data, size_t size) {

  // As the function we want to fuzz expect two integers and one string we
  // have to convert/cast the given input data into the expected variables/data
  // types
  FuzzedDataProvider fuzzed_data(data, size);
  int a = fuzzed_data.ConsumeIntegral<int>();
  int b = fuzzed_data.ConsumeIntegral<int>();
  std::string c = fuzzed_data.ConsumeRandomLengthString();

  exploreMe(a, b, c);
}
