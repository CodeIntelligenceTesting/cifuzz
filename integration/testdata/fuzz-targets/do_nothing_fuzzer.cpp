#include <stdint.h>

#include <string>

// Simple fuzz target which does not find a crash.
extern "C" int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) {
  return 0;
}
