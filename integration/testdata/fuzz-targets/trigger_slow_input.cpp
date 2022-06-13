#include <stddef.h>

#ifdef _WIN32
#include <windows.h>
#else
#include <unistd.h>
#endif

extern "C" int LLVMFuzzerTestOneInput(const char *data, size_t len) {
  // Don't show the undefined behavior if the input is empty, so that
  // we can test that the fuzzer correctly emits the test input that
  // caused the undefined behavior.
  if (len == 0)
    return 0;

#ifdef _WIN32
  Sleep(2000);
#else
  sleep(2);
#endif

  return 0;
}
