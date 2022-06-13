#include <stddef.h>

extern "C" int LLVMFuzzerTestOneInput(const char *data, size_t len) {
  // Don't trigger the timeout if the input is empty, so that we can
  // test that the fuzzer correctly emits the test input that caused the
  // timeout.
  if (len == 0) return 0;

  while (1)
    ;
  return 0;
}

