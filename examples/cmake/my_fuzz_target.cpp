#include "src/explore_me.h"
#include <cifuzz/cifuzz.h>
#include <iostream>
#include <stdio.h>
using namespace std;

FUZZ_TEST_SETUP() {}

FUZZ_TEST(const uint8_t *data, size_t size) {
  int a = 0, b = 0;
  string c = "";

  // data is an array of bytes provided by the fuzzer. As the function we want to fuzz expect
  // two integers and one string we have to convert/cast the given input data into the
  // expected variables/data types
  if (size >= 4) {
    // converting the first 4 bytes to an integer
    a = data[0] | (data[1] << 8) | (data[2] << 16) | (data[3] << 24);

    if (size >= 8) {
      // converting the next 4 bytes to an integer
      b = data[4] | (data[5] << 8) | (data[6] << 16) | (data[7] << 24);

      if (size >= 12) {
        // cast everything except the first 8 bytes to a string
        c = string(reinterpret_cast<const char *>(data + 8), size - 8);
      }
    }
  }

  exploreMe(a, b, c);
}
