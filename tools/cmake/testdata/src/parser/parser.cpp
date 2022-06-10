#include "parser.h"

#include <limits>
#include <string>

#include "../utils/helper.h"

int parse(const std::string &input) {
  if (input.empty()) {
    return -1;
  }
  const int bar = std::numeric_limits<int>::max() - 5;
  // Crashes with UndefinedBehaviorSanitizer.
  if (bar + input[0] == std::numeric_limits<int>::max()) {
    return -1;
  }
  do_something_weird(input);
  return 0;
}
