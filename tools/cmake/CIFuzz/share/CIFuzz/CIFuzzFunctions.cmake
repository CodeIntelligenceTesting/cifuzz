function(enable_fuzz_testing)

  if(CIFUZZ_ENGINE STREQUAL libfuzzer)
    if(MSVC)
      add_compile_options(/fsanitize=fuzzer-no-link)
    else()
      add_compile_options(-fsanitize=fuzzer-no-link)
    endif()
  endif()

  foreach(sanitizer IN LISTS CIFUZZ_SANITIZERS)
    if(sanitizer STREQUAL address)
      if(MSVC)
        add_compile_options(
            /fsanitize=address
            # Create a separate .pdb with debugging info used for more informative ASan reports.
            /Zi
        )
      else()
        add_compile_options(-fsanitize=address)
        add_link_options(-fsanitize=address)
      endif()
    elseif(sanitizer STREQUAL undefined)
      if(MSVC)
        message(FATAL_ERROR "CIFuzz: MSVC does not support UndefinedBehaviorSanitizer yet")
      else()
        add_compile_options(-fsanitize=undefined)
        add_link_options(-fsanitize=undefined)
      endif()
    else()
      message(FATAL_ERROR "CIFuzz: Unsupported value in CIFUZZ_SANITIZER: ${sanitizer}")
    endif()
  endforeach()
endfunction()

function(add_fuzz_test name)
  set(_options)
  set(_one_value_args)
  set(_multi_value_args)
  cmake_parse_arguments(PARSE_ARGV 1 _args "${_options}" "${_one_value_args}" "${_multi_value_args}")

  set(_args_sources ${_args_UNPARSED_ARGUMENTS})

  add_executable("${name}" ${_args_sources})
  target_include_directories("${name}" SYSTEM PRIVATE "${CIFUZZ_INCLUDE_DIR}")
  # This macro is consumed by cifuzz.h.
  target_compile_definitions("${name}" PRIVATE CIFUZZ_TEST_NAME="${name}")

  if(CIFUZZ_ENGINE STREQUAL replayer)
    target_link_libraries("${name}" PRIVATE CIFuzz_Replayer)
  elseif(CIFUZZ_ENGINE STREQUAL libfuzzer)
    if(MSVC)
      target_link_options("${name}" PRIVATE /fsanitize=fuzzer)
    elseif(CMAKE_CXX_COMPILER_ID STREQUAL "Clang")
      target_link_options("${name}" PRIVATE -fsanitize=fuzzer)
    else()
      message(FATAL_ERROR "CIFuzz: ${CMAKE_CXX_COMPILER_ID} is not supported with the libfuzzer engine")
    endif()
  else()
    message(FATAL_ERROR "CIFuzz: Unsupported value for CIFUZZ_ENGINE: ${CIFUZZ_ENGINE}")
  endif()

  set(_seed_corpus "${CMAKE_CURRENT_SOURCE_DIR}/${name}_seed_corpus")
  set(_regression_test_name "${name}_regression_test")
  if(IS_DIRECTORY "${_seed_corpus}")
    add_test(NAME "${_regression_test_name}" COMMAND "${name}" "${_seed_corpus}")
  else()
    add_test(NAME "${_regression_test_name}" COMMAND "${name}")
  endif()
endfunction()
