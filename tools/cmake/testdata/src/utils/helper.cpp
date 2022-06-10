#include "helper.h"

#include <string>

#include "../parser/secrets.h"

int do_something_weird(const std::string &input) {
  if (input.size() >= 3 && input[0] == 'a' && input[1] == 'b' && input[2] == 'c') {
    if (input.find(SECRET_VALUE) != std::string::npos) {
      char *some_string = static_cast<char *>(malloc(4));
      free(some_string);
      // Crashes with AddressSanitizer, but should not crash without it: The
      // allocated memory is addressable, but has been freed before the access.
      return some_string[1];
    }
  }
  return -1;
}
