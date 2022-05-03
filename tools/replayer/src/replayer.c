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
#include <assert.h>
#include <stdio.h>
#include <stdlib.h>

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

int main(int argc, char **argv) {
  int i;
  int res;

  fprintf(stderr, "StandaloneFuzzTargetMain: running %d inputs\n", argc - 1);
  LLVMFuzzerInitializeIfPresent(&argc, &argv);
  for (i = 1; i < argc; i++) {
    FILE *f;
    size_t len;
    unsigned char *buf;
    size_t n_read;

    fprintf(stderr, "Running: %s\n", argv[i]);
#ifdef _WIN32
    /* fopen is deprecated in the Microsoft CRT. */
    fopen_s(&f, argv[i], "r");
#else
    /* fopen_s is not available in Unix C90 system headers. */
    f = fopen(argv[i], "r");
#endif
    if (f == NULL) {
      perror("Failed to open file");
      exit(1);
    }
    fseek(f, 0, SEEK_END);
    len = ftell(f);
    fseek(f, 0, SEEK_SET);
    buf = (unsigned char*)malloc(len);
    n_read = fread(buf, 1, len, f);
    fclose(f);
    assert(n_read == len);
    res = LLVMFuzzerTestOneInput(buf, len);
    /* Avoid "unused but set variable" warnings if asserts are compiled out with NDEBUG. */
    (void)res;
    assert(res == 0);
    free(buf);
    fprintf(stderr, "Done:    %s: (%ld bytes)\n", argv[i], (unsigned long) n_read);
  }
  return 0;
}
