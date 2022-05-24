#include <iostream>

#include "parser.h"
#include <cifuzz/cifuzz.h>

FUZZ_TEST_SETUP() {
  std::cout << "FUZZ_TEST_SETUP ran" << std::endl;
}

FUZZ_TEST(const std::uint8_t *data, std::size_t size) {
  parse(std::string(reinterpret_cast<const char*>(data), size));
}
