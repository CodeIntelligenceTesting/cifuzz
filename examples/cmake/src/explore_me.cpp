#include "explore_me.h"
#include <cstdio>
#include <string.h>
using namespace std;

// just a function with multiple paths that can be discoverd by a fuzzer
void exploreMe(int a, int b, string c) {
  printf("a: %d; b: %d; c: %s\n", a, b, c.c_str());

  if (a >= 20000) {
    printf("branch 1\n");

    if (b >= 2000000) {
      printf("branch 2\n");

      if (b - a < 100000) {
        printf("branch 3\n");

        if (c == "FUZZING") {
          printf("branch 4\n");

          // Trigger a heap buffer overflow
          char *s = (char *)malloc(1);
          strcpy(s, "too long");
          printf("%s\n", s);
        }
      }
    }
  } else {
    printf("this is the default path\n");
  }
  printf("---------\n");
}
