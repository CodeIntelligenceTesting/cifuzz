#include <stddef.h>
#include <stdlib.h>

extern "C" int LLVMFuzzerTestOneInput(const char *data, size_t len) {
  // Don't trigger the OOM if the input is empty, so that we can test
  // that the fuzzer correctly emits the test input that caused the OOM.
  if (len == 0) return 0;

  char *ptr;
  long i;

  // Allocate exponentially growing chunks of memory until OOM
  i = 1024L;
  while (true) {
    ptr = (char *)malloc(i);
    i *= 2;
  }

  return 0;
}
