#include <assert.h>

#include <cifuzz/cifuzz.h>

FUZZ_TEST_SETUP() {}

FUZZ_TEST(const uint8_t *data, size_t size) {
  if (size > 0) {
    assert(data[0] == data[0]);
  }
}
