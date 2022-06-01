#include <stdint.h>

#include <string>

// Simple fuzz target which does not find a crash but a few new paths
extern "C" int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) {
  if (size == 2) {
    if (data[0] == 1 && data[1] == 0) {
      return 0;
    }
  }
  return 0;
}
