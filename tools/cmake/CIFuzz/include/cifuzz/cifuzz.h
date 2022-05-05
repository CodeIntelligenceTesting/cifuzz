#include <cstddef>
#include <cstdint>

#define FUZZ_TEST                                                         \
void LLVMFuzzerTestOneInputNoReturn(const uint8_t *data, size_t size);    \
extern "C" int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) { \
  LLVMFuzzerTestOneInputNoReturn(data, size);                             \
  return 0;                                                               \
}                                                                         \
void LLVMFuzzerTestOneInputNoReturn

#define FUZZ_TEST_SETUP \
void LLVMFuzzerInitializeNoReturn();    \
extern "C" int LLVMFuzzerInitialize(int *argc, char ***argv) { \
  LLVMFuzzerInitializeNoReturn();                              \
  return 0;                                                    \
}                                                              \
void LLVMFuzzerInitializeNoReturn

