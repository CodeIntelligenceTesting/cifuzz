#ifndef CIFUZZ_CIFUZZ_H
#define CIFUZZ_CIFUZZ_H

/* Include the headers providing the definitions required to use FUZZ_TEST. */
#ifdef __cplusplus
#include <cstddef>
#include <cstdint>
#else
#include <stddef.h>
#include <stdint.h>
#endif

#if defined(__CLION_IDE__) && defined(__cplusplus)
/* This code will only be seen by CLion's static analysis/preprocessing engine
 * and thus doesn't have to contain any definitions, declarations are
 * sufficient. It mocks enough of the Doctest classes to make CLion's test
 * framework support treat it as the full Doctest library. */
namespace doctest {
namespace detail {
struct TestSuite {};
typedef int (*funcType)(const uint8_t *data, std::size_t size);
struct TestCase
{
  TestCase(funcType test, const char* file, unsigned line, const TestSuite& test_suite,
           const char* type = "", int template_id = -1);
  TestCase& operator*(const char* in);
};
int regTest(const TestCase& tc);
}
}

/* This macro has to be defined or CLion will not show a play button, but the
 * value doesn't matter. */
#define DOCTEST_TEST_CASE

#define CLION_TEST_PLAY_BUTTON \
/* Silence a CLion warning about a static
 * initializer with static storage duration */  \
/* NOLINTBEGIN(cert-err58-cpp) */               \
static const int DOCTEST_ANON_VAR_15771531 =    \
    doctest::detail::regTest(                   \
        doctest::detail::TestCase(              \
            &LLVMFuzzerTestOneInput,            \
            "",                                 \
            1,                                  \
            doctest::detail::TestSuite()        \
/* This string is used as the test name and has
 * to be globally unique so that CLion
 * generates a unique run configuration per
 * test. */                                     \
        ) * CIFUZZ_TEST_NAME);                  \
/* NOLINTEND(cert-err58-cpp) */
#else
#define CLION_TEST_PLAY_BUTTON
#endif

#ifdef __cplusplus
#define CIFUZZ_C_LINKAGE extern "C"
#else
#define CIFUZZ_C_LINKAGE
#endif

#ifndef CIFUZZ_TEST_NAME
#define CIFUZZ_TEST_NAME NULL
#endif
#ifndef CIFUZZ_SEED_CORPUS
#define CIFUZZ_SEED_CORPUS NULL
#endif

#define FUZZ_TEST                                                                \
static void LLVMFuzzerTestOneInputNoReturn(const uint8_t *data, size_t size);    \
CIFUZZ_C_LINKAGE int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) {  \
  LLVMFuzzerTestOneInputNoReturn(data, size);                                    \
  return 0;                                                                      \
}                                                                                \
CIFUZZ_C_LINKAGE const char *cifuzz_test_name(void) {                            \
  return CIFUZZ_TEST_NAME;                                                       \
}                                                                                \
CIFUZZ_C_LINKAGE const char *cifuzz_seed_corpus(void) {                          \
  return CIFUZZ_SEED_CORPUS;                                                     \
}                                                                                \
CLION_TEST_PLAY_BUTTON                                                           \
void LLVMFuzzerTestOneInputNoReturn

#define FUZZ_TEST_SETUP                                              \
static void LLVMFuzzerInitializeNoReturn(void);                      \
CIFUZZ_C_LINKAGE int LLVMFuzzerInitialize(int *argc, char ***argv) { \
  (void) argc;                                                       \
  (void) argv;                                                       \
  LLVMFuzzerInitializeNoReturn();                                    \
  return 0;                                                          \
}                                                                    \
void LLVMFuzzerInitializeNoReturn

#endif  // CIFUZZ_CIFUZZ_H
