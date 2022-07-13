add_library(cifuzz_internal_replayer STATIC EXCLUDE_FROM_ALL
            "${CMAKE_CURRENT_LIST_DIR}/../src/replayer.c")
# If a CXX-only project depends on the replayer, it won't be able to compile and link the replayer's C source file.
# Using enable_language(C) from a package is discouraged, but we can work around this by marking the replayer as a CXX
# target if C is not enabled - it should build just fine with any C++ compiler.
# https://discourse.cmake.org/t/is-it-appropriate-to-use-enable-language-in-a-cmake-package-file/4335/2
if(NOT C IN_LIST ENABLED_LANGUAGES)
  set_target_properties(cifuzz_internal_replayer PROPERTIES LINKER_LANGUAGE CXX)
endif()
if(CIFUZZ_SANITIZERS)
  target_compile_definitions(cifuzz_internal_replayer PRIVATE CIFUZZ_HAS_SANITIZER)
endif()
