#include <stddef.h>
#include <stdint.h>

extern "C" int FUZZ_INIT() {
  return 0;
}

extern "C" int FUZZ(const uint8_t *Data, size_t Size) {
  return 0;
}
