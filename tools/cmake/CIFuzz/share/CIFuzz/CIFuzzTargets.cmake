add_library(CIFuzz_Replayer STATIC EXCLUDE_FROM_ALL
            "${CMAKE_CURRENT_LIST_DIR}/../../src/replayer.c")
if(CIFUZZ_SANITIZERS)
  target_compile_definitions(CIFuzz_Replayer PRIVATE CIFUZZ_HAS_SANITIZER)
endif()
