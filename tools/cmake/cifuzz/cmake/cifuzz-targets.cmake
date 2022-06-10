add_library(cifuzz_internal_replayer STATIC EXCLUDE_FROM_ALL
            "${CMAKE_CURRENT_LIST_DIR}/../src/replayer.c")
if(CIFUZZ_SANITIZERS)
  target_compile_definitions(cifuzz_internal_replayer PRIVATE CIFUZZ_HAS_SANITIZER)
endif()
