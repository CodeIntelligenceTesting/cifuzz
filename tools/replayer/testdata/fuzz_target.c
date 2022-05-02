#include <assert.h>
#include <limits.h>
#include <stdio.h>

/* volatile to prevent compiler optimizations, global to prevent unused-but-set-variable warnings */
static volatile int some_int = INT_MAX;

#ifndef DISABLE_FUZZER_INITIALIZE
int LLVMFuzzerInitialize(int *argc, char ***argv) {
  printf("init(%d,%s)\n", *argc, (*argv)[0]);
  return 0;
}
#endif

/*
 * This fuzz target behaves as follows on these particular inputs:
 *   - 'asan': Produces an ASan finding.
 *   - 'ubsan': Produces a UBSan finding.
 *   - 'assert': Fails an assert.
 *   - 'return': Returns a non-zero value.
 *   - all other values: Prints the input to stdout interpreted as ASCII,
 *                       wrapped in single quotes and followed by a newline.
 */
int LLVMFuzzerTestOneInput(const unsigned char *data, size_t size) {
  size_t i;

  if (size == 4 && data[0] == 'a' && data[1] == 's' && data[2] == 'a' && data[3] == 'n') {
    /* Out-of-bounds read (detected by ASan). */
    some_int = data[4];
  } else if (size == 5 && data[0] == 'u' && data[1] == 'b' && data[2] == 's' && data[3] == 'a' && data[4] == 'n') {
    /* Shift by 'n' (110) exceeds bit-width of n (detected by UBSan). */
    some_int <<= data[4];
  } else if (size == 6 && data[0] == 'a' && data[1] == 's' && data[2] == 's' && data[3] == 'e' && data[4] == 'r'
      && data[5] == 't') {
    assert(0);
  } else if (size == 6 && data[0] == 'r' && data[1] == 'e' && data[2] == 't' && data[3] == 'u' && data[4] == 'r'
      && data[5] == 'n') {
    return 1;
  }

  putchar('\'');
  for (i = 0; i < size; i++) {
    putchar(data[i]);
  }
  putchar('\'');
  putchar('\n');
  /* Ensure that all output has been written in case the next execution crashes. */
  fflush(stdout);
  return 0;
}
