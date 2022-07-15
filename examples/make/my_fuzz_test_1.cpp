#include <stdint.h>

#include <string>

#include "api.h"

extern "C" int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) {
  DoStuff(std::string((char *)data, size));
  return 0;
}
