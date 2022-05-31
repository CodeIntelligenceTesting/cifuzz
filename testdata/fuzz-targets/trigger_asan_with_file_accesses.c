#include <errno.h>
#include <fcntl.h>
#ifndef _WIN32
#include <pwd.h>
#endif
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>

void AssertFileAccess(const char *pathname, int flags) {
  int fd = open(pathname, flags);
  if (fd < 0) {
    fprintf(stderr, "##### ASSERTION FAILED ##### Error accessing %s: %s\n",
            pathname, strerror(errno));
    exit(1);
  }
}

// Fuzz target which accesses some files which should be accessible to
// fuzz targets and then calls DoStuff().
int LLVMFuzzerTestOneInput(const char *data, size_t len) {
  // Check file accesses
  // The config file from the current working directory should be readable
  AssertFileAccess("example.conf", O_RDONLY);

#ifndef _WIN32
  AssertFileAccess("/dev/urandom", O_RDONLY);
  AssertFileAccess("/dev/null", O_RDONLY);

  // Check that we can obtain the UID for user "root"
  struct passwd *pw = getpwnam("root");
  if (pw == NULL) {
    fprintf(
        stderr,
        "##### ASSERTION FAILED ##### Error obtaining UID of user root: %s\n",
        strerror(errno));
    exit(1);
  }
#endif

  // Don't crash if the input is empty, so that we can test that the
  // fuzzer correctly emits the test input that caused the crash.
  if (len == 0)
    return 0;

  // Also don't crash if the input is the AFL input seed, because AFL
  // refuses to run without input seeds that don't crash the target
  if (len == 1 && data[0] == '?')
    return 0;

  // Trigger a heap buffer overflow
  char *s = (char *)malloc(1);
  strcpy(s, "too long");
  printf("%s\n", s);

  return 0;
}
