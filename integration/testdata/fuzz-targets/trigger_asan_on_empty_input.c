#include <stdint.h>
#include <string.h>

int LLVMFuzzerTestOneInput(const char *data, size_t len) {
  *(char *)0xdeadbeef = 0;
  return 0;
}
