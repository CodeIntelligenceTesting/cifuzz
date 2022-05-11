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
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>

#if defined(_WIN32)
#define POSIX_STAT _stat
#define POSIX_S_IFDIR _S_IFDIR
#define POSIX_S_IFREG _S_IFREG
#include <windows.h>
#else
#define POSIX_STAT stat
#define POSIX_S_IFDIR S_IFDIR
#define POSIX_S_IFREG S_IFREG
#include <dirent.h>
#endif

extern int LLVMFuzzerTestOneInput(const unsigned char *data, size_t size);

#if defined(_WIN32)
/*
 * MSVC does not support weak symbols, but /alternatename can be used to direct the linker to a default implementation.
 * https://github.com/llvm/llvm-project/blob/0c8c05064d57fe3bbbb1edd4c6e67f909c720578/compiler-rt/lib/fuzzer/FuzzerExtFunctionsWindows.cpp
 */
int LLVMFuzzerInitializeDefault(int *argc, char ***argv) {
  (void)argc;
  (void)argv;
  return 0;
}

#if defined(_MSC_VER)
#pragma comment(linker, "/alternatename:LLVMFuzzerInitialize=LLVMFuzzerInitializeDefault")
#else
__attribute__((weak, alias("LLVMFuzzerInitializeDefault")))
#endif
int LLVMFuzzerInitialize(int *argc, char ***argv);

static void LLVMFuzzerInitializeIfPresent(int *argc, char ***argv) {
  LLVMFuzzerInitialize(argc, argv);
}
#elif defined(__APPLE__)
/*
 * Weak symbols require specifying -U on the command line on macOS, hence use dlsym to find LLVMFuzzerInitialize.
 * https://github.com/llvm/llvm-project/blob/0c8c05064d57fe3bbbb1edd4c6e67f909c720578/compiler-rt/lib/fuzzer/FuzzerExtFunctionsDlsym.cpp
 */
#include <dlfcn.h>

static void LLVMFuzzerInitializeIfPresent(int *argc, char ***argv) {
  void *fn_ptr = dlsym(RTLD_DEFAULT, "LLVMFuzzerInitialize");
  if (fn_ptr != NULL) {
    ((int (*)(int *, char ***)) fn_ptr)(argc, argv);
  }
}
#else
/*
 * General Unix is assumed to have support for weak symbols, but doesn't export symbols dynamically by default, which
 * precludes using the dlsym approach.
 * https://github.com/llvm/llvm-project/blob/0c8c05064d57fe3bbbb1edd4c6e67f909c720578/compiler-rt/lib/fuzzer/FuzzerExtFunctionsWeak.cpp
 */
__attribute__((weak)) int LLVMFuzzerInitialize(int *argc, char ***argv) {
  (void)argc;
  (void)argv;
  return 0;
}
static void LLVMFuzzerInitializeIfPresent(int *argc, char ***argv) {
  LLVMFuzzerInitialize(argc, argv);
}
#endif

void run_one_input(const unsigned char *data, size_t size) {
  int res;

  res = LLVMFuzzerTestOneInput(data, size);
  /* Avoid "unused but set variable" warnings if asserts are compiled out with NDEBUG. */
  (void)res;
  assert(res == 0);
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

int main(int argc, char **argv) {
  int i;
  unsigned char empty[1];

  LLVMFuzzerInitializeIfPresent(&argc, &argv);

  fprintf(stderr, "Running: <empty input>\n");
  run_one_input(&empty[0], 0);
  fprintf(stderr, "Done:    <empty input>: (0 bytes)\n");

  for (i = 1; i < argc; i++) {
    run_file_or_dir(argv[i]);
  }
  return 0;
}
