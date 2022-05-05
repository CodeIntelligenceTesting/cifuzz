#include <cassert>

#include <cifuzz/cifuzz.h>

bool should_crash = false;

FUZZ_TEST_SETUP() {
  should_crash = true;
}

// This fuzz target crashes on any input if and only if FUZZ_TEST_SETUP has been
// invoked.
FUZZ_TEST(const uint8_t *data, size_t size) {
  if (should_crash) {
    assert(0 == 1);
  }
}
