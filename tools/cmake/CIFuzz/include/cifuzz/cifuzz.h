#include <cstddef>
#include <cstdint>

#ifdef __CLION_IDE__
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

#define FUZZ_TEST                                                         \
void LLVMFuzzerTestOneInputNoReturn(const uint8_t *data, size_t size);    \
extern "C" int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) { \
  LLVMFuzzerTestOneInputNoReturn(data, size);                             \
  return 0;                                                               \
}                                                                         \
CLION_TEST_PLAY_BUTTON                                                    \
void LLVMFuzzerTestOneInputNoReturn

#define FUZZ_TEST_SETUP \
void LLVMFuzzerInitializeNoReturn();    \
extern "C" int LLVMFuzzerInitialize(int *argc, char ***argv) { \
  LLVMFuzzerInitializeNoReturn();                              \
  return 0;                                                    \
}                                                              \
void LLVMFuzzerInitializeNoReturn

