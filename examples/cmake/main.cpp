#include "src/explore_me.h"

int main() {
  exploreMe(1, 1, "A");
  exploreMe(2147483647, 1, "A");
  exploreMe(2147483647, 2147483647, "A");
  exploreMe(2000000000, 2000000123, "A");
  exploreMe(2000000000, 2000000123, "FUZZ");
}
