/*
 * Based on:
 * https://github.com/llvm/llvm-project/blob/95fedfab6cfb82a2fe1010d266b1269425f5eb46/compiler-rt/lib/fuzzer/standalone/StandaloneFuzzTargetMain.c
 *
 * Modified by Fabian Meumertzheim:
 *   - moved variable declarations to conform to ANSI C90 standard
 *   - added explicit return to conform to ANSI C90 standard
 *   - replaced %zd with %ld to conform to ANSI C90 standard
 *   - replaced fopen with fopen_s on Windows and added an error message
 *   - added assert on LLVMFuzzerTestOneInput return value to mimic libFuzzer
 *   - made weak semantics of LLVMFuzzerInitialize work on macOS and Windows via
 *     dlsym and /alternatename
 *   - added an assert on malloc return value
 *   - added support for specifying directories as inputs
 *   - always run on the empty input
 *   - crash on UBSan findings
 *   - disabled dialog boxes for aborts or failed asserts on Windows
 *   - emit summary and follow-up suggestions at the end
 */
/*===- StandaloneFuzzTargetMain.c - standalone main() for fuzz targets. ---===//
//
// Part of the LLVM Project, under the Apache License v2.0 with LLVM Exceptions.
// See https://llvm.org/LICENSE.txt for license information.
// SPDX-License-Identifier: Apache-2.0 WITH LLVM-exception
//
//===----------------------------------------------------------------------===//
// This main() function can be linked to a fuzz target (i.e. a library
// that exports LLVMFuzzerTestOneInput() and possibly LLVMFuzzerInitialize())
// instead of libFuzzer. This main() function will not perform any fuzzing
// but will simply feed all input files one by one to the fuzz target.
//
// Use this file to provide reproducers for bugs when linking against libFuzzer
// or other fuzzing engine is undesirable.
//===----------------------------------------------------------------------===*/
#if defined(__linux__)
/* Using S_IFDIR with gcc's -ansi requires this define. */
#define _XOPEN_SOURCE 700
#endif
#include <assert.h>
#include <errno.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>

#if defined(_WIN32)
#define POSIX_STAT _stat
#define POSIX_S_IFDIR _S_IFDIR
#define POSIX_S_IFREG _S_IFREG
#include <windows.h>
#include <crtdbg.h>
#else
#define POSIX_STAT stat
#define POSIX_S_IFDIR S_IFDIR
#define POSIX_S_IFREG S_IFREG
#include <dirent.h>
#endif

static const char *argv0;
static int all_inputs_passed = 0;
static int num_passing_inputs = 0;
static const char *current_input = NULL;

/* Keep in sync with strsignal below. */
static const int TERMINATING_SIGNALS[] = {
    SIGABRT,
    SIGFPE,
    SIGILL,
    SIGSEGV,
    SIGTERM,
};
static const int NUM_TERMINATING_SIGNALS = sizeof(TERMINATING_SIGNALS) / sizeof(int);

#ifdef _WIN32
/* The Microsoft C Runtime lacks strsignal, so we provide our own implementation for the signals we are handling. */
static const char *strsignal(int sig) {
  switch(sig) {
  case SIGABRT:
    return "Aborted";
  case SIGFPE:
    return "Arithmetic exception";
  case SIGILL:
    return "Illegal instruction";
  case SIGSEGV:
    return "Segmentation fault";
  case SIGTERM:
    return "Terminated";
  default:
    return NULL;
  }
}
#endif

const char *__ubsan_default_options() {
  /*
   * With the reproducer, UBSan findings should always be fatal so that they lead to a non-zero exit code.
   */
  return "halt_on_error=1";
}

extern int LLVMFuzzerTestOneInput(const unsigned char *data, size_t size);

#define STRINGIFY_(x) #x
#define STRINGIFY(x) STRINGIFY_(x)

/*
 * The following macros provide a cross-platform way of referencing an optional symbol with a provided default.
 * DEFINE_DEFAULT defines the default implementation and WITH_DEFAULT is a function pointer that points to the real
 * symbol if it is defined and to the default implementation otherwise.
 */
#if defined(_WIN32)
/*
 * MSVC does not support weak symbols, but /alternatename can be used to direct the linker to a default implementation.
 * https://github.com/llvm/llvm-project/blob/0c8c05064d57fe3bbbb1edd4c6e67f909c720578/compiler-rt/lib/fuzzer/FuzzerExtFunctionsWindows.cpp
 */
#if defined(_MSC_VER)
#define DEFINE_DEFAULT(ret, name, args)                                                   \
ret name##Default args;                                                                   \
__pragma(comment(linker, "/alternatename:" STRINGIFY(name) "=" STRINGIFY(name##Default))) \
ret name args;                                                                            \
ret name##Default args
#else
#define DEFINE_DEFAULT(ret, name, args)                \
ret name##Default args;                                \
__attribute__((weak, alias(STRINGIFY(name##Default)))) \
ret name args;                                         \
ret name##Default args
#endif

#define WITH_DEFAULT(name) name

#elif defined(__APPLE__)
/*
 * Weak symbols require specifying -U on the command line on macOS, hence use dlsym to find LLVMFuzzerInitialize.
 * https://github.com/llvm/llvm-project/blob/0c8c05064d57fe3bbbb1edd4c6e67f909c720578/compiler-rt/lib/fuzzer/FuzzerExtFunctionsDlsym.cpp
 */
#include <dlfcn.h>

#define DEFINE_DEFAULT(ret, name, args)                \
static ret name##Default args;                         \
static ret (*name##Ptr()) args {                       \
  void *fn_ptr = dlsym(RTLD_DEFAULT, STRINGIFY(name)); \
  if (fn_ptr != NULL) {                                \
    return (ret (*) args) fn_ptr;                      \
  }                                                    \
  return name##Default;                                \
}                                                      \
static ret name##Default args

#define WITH_DEFAULT(name) name##Ptr()

#else
/*
 * General Unix is assumed to have support for weak symbols, but doesn't export symbols dynamically by default, which
 * precludes using the dlsym approach.
 * https://github.com/llvm/llvm-project/blob/0c8c05064d57fe3bbbb1edd4c6e67f909c720578/compiler-rt/lib/fuzzer/FuzzerExtFunctionsWeak.cpp
 */
#define DEFINE_DEFAULT(ret, name, args) __attribute__((weak)) ret name args
#define WITH_DEFAULT(name) name

#endif

DEFINE_DEFAULT(int, LLVMFuzzerInitialize, (int *argc, char ***argv)) {
  (void)argc;
  (void)argv;
  return 0;
}

/* Set by the FUZZ_TEST macro defined in cifuzz.h. */
DEFINE_DEFAULT(const char*, cifuzz_test_name, (void)) {
  return NULL;
}

/* Set by the CMake integration if a sanitizer is linked in.
 * Detecting this via compiler macros isn't possible since gcc does not define a macro for UBSan.
 * Using DEFINE_DEFAULT/WITH_DEFAULT isn't possible since sanitizer runtimes are usually linked dynamically and thus
 * don't override weak symbols.
 * TODO: If this ever becomes an issue with other build systems, replace the compile-time check with a dynamic lookup
 *       at runtime. */
#ifdef CIFUZZ_HAS_SANITIZER
void __sanitizer_set_death_callback(void (*callback)(void));
#endif

void run_one_input(const unsigned char *data, size_t size) {
  int res;

  res = LLVMFuzzerTestOneInput(data, size);
  /* Avoid "unused but set variable" warnings if asserts are compiled out with NDEBUG. */
  (void)res;
  assert(res == 0);
  num_passing_inputs++;
}

void run_file(const char *path) {
  FILE *f;
  size_t len;
  unsigned char *buf;
  size_t n_read;

  fprintf(stderr, "Running: %s\n", path);
#ifdef _WIN32
  /* fopen is deprecated in the Microsoft CRT. */
    fopen_s(&f, path, "r");
#else
  /* fopen_s is not available in Unix C90 system headers. */
  f = fopen(path, "r");
#endif
  if (f == NULL) {
    perror("Failed to open file");
    exit(1);
  }
  fseek(f, 0, SEEK_END);
  len = ftell(f);
  fseek(f, 0, SEEK_SET);
  buf = (unsigned char*)malloc(len);
  assert(buf != NULL);
  n_read = fread(buf, 1, len, f);
  fclose(f);
  assert(n_read == len);
  current_input = path;
  run_one_input(buf, len);
  free(buf);
  fprintf(stderr, "Done:    %s: (%ld bytes)\n", path, (unsigned long) n_read);
}

void run_file_or_dir(const char *path);

void run_dir_entry(const char *dir, const char *file) {
  char *path;
  size_t path_size;

  /* Skip hidden files as well as "." and "..". File names can't be empty. */
  if (file[0] == '.') {
    return;
  }

  /* +1 for path separator, +1 for "\0". */
  path_size = strlen(dir) + 1 + strlen(file) + 1;
  path = (char*)malloc(path_size);
  assert(path != NULL);
#ifdef _WIN32
  /* sprintf is deprecated in the Microsoft CRT. */
  sprintf_s(path, path_size, "%s\\%s", dir, file);
#else
  /* sprintf_s is not available in Unix C90 system headers. */
  sprintf(path, "%s/%s", dir, file);
#endif
  run_file_or_dir(path);
  free(path);
}

void traverse_dir(const char *path) {
#if defined(_WIN32)
  WIN32_FIND_DATA fd;
  HANDLE h_find;
  char *filter;
  size_t filter_size;

  /* +2 for "\*", +1 for "\0". */
  filter_size = strlen(path) + 3;
  filter = (char*)malloc(filter_size);
  assert(filter != NULL);
  sprintf_s(filter, filter_size, "%s\\*", path);
  if ((h_find = FindFirstFile(filter, &fd)) == INVALID_HANDLE_VALUE) {
    /* TODO: Include the stringified last error in the message. */
    fprintf(stderr, "Failed to list files in '%s'\n", path);
    exit(1);
  }
  do {
    run_dir_entry(path, fd.cFileName);
  } while (FindNextFile(h_find, &fd) != 0);
  if (GetLastError() != ERROR_NO_MORE_FILES) {
    /* TODO: Include the stringified last error in the message. */
    fprintf(stderr, "Failed to list files in '%s'\n", path);
    exit(1);
  }
  FindClose(h_find);
  free(filter);
#else
  DIR *dir;
  struct dirent *dir_entry;

  dir = opendir(path);
  if (dir == NULL) {
    fprintf(stderr, "Failed to open directory '%s': %s\n", path, strerror(errno));
    exit(1);
  }
  errno = 0;
  while ((dir_entry = readdir(dir)) != NULL) {
    run_dir_entry(path, dir_entry->d_name);
  }
  if (errno != 0) {
    fprintf(stderr, "Failed to list files in '%s': %s\n", path, strerror(errno));
    exit(1);
  }
  closedir(dir);
#endif
}

void run_file_or_dir(const char *path) {
  int res;
  struct POSIX_STAT stat_info;

  res = POSIX_STAT(path, &stat_info);
  if (res != 0) {
    fprintf(stderr, "Failed to access '%s': ", path);
    /* strerror is deprecated in the Microsoft CRT. */
    perror("");
    exit(1);
  }
  if (stat_info.st_mode & POSIX_S_IFDIR) {
    traverse_dir(path);
  } else if (stat_info.st_mode & POSIX_S_IFREG) {
    run_file(path);
  } else {
    fprintf(stderr, "File type of '%s' is unsupported: %d\n", path, stat_info.st_mode);
    exit(1);
  }
}

static void print_summary(const char *failure_reason) {
  if (all_inputs_passed) {
    fprintf(stderr, "\nRan fuzz test on %d inputs - passed\n\n"
                    "Note: No fuzzing has been performed, the fuzz test has only been executed on the\n"
                    "fixed inputs in the seed corpus.\n\n", num_passing_inputs);
    if (WITH_DEFAULT(cifuzz_test_name)()) {
      fprintf(stderr, "To start a fuzzing run, execute:\n\n"
                      "    cifuzz run %s\n", WITH_DEFAULT(cifuzz_test_name)());
    }
  } else {
    if (current_input) {
      fprintf(stderr, "\nFuzz test failed on input '%s'\n"
                      "Reason: %s\n\n", current_input, failure_reason);
      /* TODO: Replace with cifuzz debug on all platforms when it has been implemented. */
#ifdef __linux__
      fprintf(stderr, "To debug this failure, execute:\n\n"
                      "    gdb -ex 'break LLVMFuzzerTestOneInput' -ex run --args '%s' '%s'\n", argv0, current_input);
#endif
    } else {
      if (failure_reason == NULL) {
        failure_reason = "Unknown";
      }
      /* The empty input is always executed first by the replayer, so we do not need to pass in an empty file. */
      fprintf(stderr, "\nFuzz test failed on the empty input\n"
                      "Reason: %s\n\n", failure_reason);
      /* TODO: Replace with cifuzz debug on all platforms when it has been implemented. */
#ifdef __linux__
      fprintf(stderr, "To debug this failure, execute:\n\n"
                      "    gdb -ex 'break LLVMFuzzerTestOneInput' -ex run --args '%s'\n", argv0);
#endif
    }
  }
}

static void explicit_exit_handler(void) {
  print_summary("Fuzz target exited");
}

static void terminating_signal_handler(int sig) {
  /*
   * The default action for the signals we handle is to terminate the process, so this is the time to write our report
   * and explain the termination. Afterwards, disable the handler and re-raise the signal to trigger the default action.
   */
  print_summary(strsignal(sig));
  signal(sig, SIG_DFL);
  raise(sig);
  return;
}

static void register_terminating_signal_handler(int sig) {
  void (*old_handler)(int sig) = signal(sig, terminating_signal_handler);
  if (old_handler == SIG_ERR) {
    /* Signal handling is best-effort, print a warning and continue. */
    fprintf(stderr, "Failed to register handler for " STRINGIFY(SIGNAL) ": error %d", errno);
    return;
  }
  /* We only want to replace the default handler since it would terminate the program in a clearly undesired way. If the
   * fuzz test registers a signal handler, we expect the relevant signal to indicate a benign situation. */
  if (old_handler != SIG_DFL) {
    /* Restore the old handler and exit.
     * Note: We wouldn't have to do this if we used sigaction instead, but it isn't available on Windows. */
    old_handler = signal(sig, old_handler);
    if (old_handler == SIG_ERR) {
      /* Failing to restore the previous signal handling behavior may affect the execution of the fuzz test and is thus
       * a fatal error. */
      fprintf(stderr, "Failed to restore handler for " STRINGIFY(SIGNAL) ": error %d", errno);
      exit(1);
    }
  }
}

#ifdef CIFUZZ_HAS_SANITIZER
static void sanitizer_report_handler(void) {
  print_summary("Sanitizer finding");
}
#endif

int main(int argc, char **argv) {
  int i;
  unsigned char empty[1];

  argv0 = argv[0];

  /* Best-effort attempt at registering handlers that run right before any kind of normal or abnormal program exit so
   * that the summary of executed inputs and follow-up commands can be printed. */
  atexit(explicit_exit_handler);
  for (i = 0; i < NUM_TERMINATING_SIGNALS; ++i) {
    register_terminating_signal_handler(TERMINATING_SIGNALS[i]);
  }
#ifdef CIFUZZ_HAS_SANITIZER
  __sanitizer_set_death_callback(sanitizer_report_handler);
#endif

#if _WIN32
  /* Disable the dialog box shown for a failed assert. */
  _CrtSetReportMode(_CRT_ASSERT, _CRTDBG_MODE_DEBUG);
#if !defined(__MSVCRT_VERSION__) || __MSVCRT_VERSION >= 0x900
  /* Using _set_abort_behavior with MinGW requires setting the MinGW-specific
   * setting __MSVCRT_VERSION__ to at least 0x900.
   * https://lists.llvm.org/pipermail/llvm-dev/2015-January/081208.html */
  /* Disable the dialog box shown for a call to abort(), but still generate a
   * crash dump. */
  _set_abort_behavior(0, _WRITE_ABORT_MSG);
#endif
#endif

  WITH_DEFAULT(LLVMFuzzerInitialize)(&argc, &argv);

  fprintf(stderr, "Running: <empty input>\n");
  run_one_input(&empty[0], 0);
  fprintf(stderr, "Done:    <empty input>: (0 bytes)\n");

  for (i = 1; i < argc; i++) {
    run_file_or_dir(argv[i]);
  }

  all_inputs_passed = 1;
  return 0;
}
