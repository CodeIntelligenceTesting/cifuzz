#include <stddef.h>

int TriggerUndefinedBehaviorSanitizer() {
  int n = 23;
  n <<= 32;
  return n;
}

extern "C" int LLVMFuzzerTestOneInput(const char *data, size_t len) {
  // Don't show the undefined behavior if the input is empty, so that
  // we can test that the fuzzer correctly emits the test input that
  // caused the undefined behavior.
  if (len == 0) return 0;

  TriggerUndefinedBehaviorSanitizer();

  return 0;
}
