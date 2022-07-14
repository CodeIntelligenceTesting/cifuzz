# Note: Keep the clang flags used below in sync with internal/cmd/run/run.go
#       Explanations of these flags are provided in that file.
function(enable_fuzz_testing)
  # Remove the metadata directory we populate with fuzz test target information at configuration time so that e.g.
  # metadata for renamed or removed targets doesn't linger around.
  file(REMOVE_RECURSE "${CMAKE_BINARY_DIR}/$<CONFIG>/.cifuzz")

  # Conceptually, "building for fuzzing" is similar to a build type such as Release or RelWithDebInfo. We instead use
  # a cache variable that adds flags to a base configuration we assume to be RelWithDebInfo for multiple reasons:
  # 1. Custom build types require defining a potentially unknown set of cache variables and are thus hard to maintain.
  # 2. Since custom build types store the flags in cache variables, cifuzz updates changing the flags would require
  #    regenerating CMake build directories rather than just building them.
  # 3. Many projects contain checks for the name of the build type, which makes us more compatible if we use an existing
  #    one.
  if(CIFUZZ_TESTING)
    add_compile_definitions(FUZZING_BUILD_MODE_UNSAFE_FOR_PRODUCTION)
    if(MSVC)
      add_compile_options(
          # Allow the compiler to inline more aggressively. This overrides the (questionable?) default of /Ob1 set by
          # CMake's RelWithDebInfo configuration (see https://stackoverflow.com/a/66089368/297261). Given that it also
          # sets /Zi, which implies /Zo, which promises that it "tells the compiler to generate additional debugging
          # information for local variables and inlined functions" (see
          # https://docs.microsoft.com/en-us/cpp/build/reference/zo-enhance-optimized-debugging?view=msvc-170).
          /Ob2
          # MSVC's equivalent of -fno-omit-frame-pointer.
          /Oy-
          # Undefine NDEBUG, which is explicitly defined by the RelWithDebInfo CMake configuration, so that asserts are
          # kept.
          /UNDEBUG
          # Link the CRT statically so that ASan is also linked statically.
          # TODO(fmeum): Remove once ASan has been stabilized and clang_rt.asan_dynamic-x86_64.dll is available in the
          #  default PATH, e.g. in System32.
          # https://stackoverflow.com/a/66532115/297261
          /MTd
      )
      add_link_options(
          # /INCREMENTAL is enabled by default with RelWithDebInfo, but is unsupported with ASan and potentially impacts
          # performance by padding functions.
          # https://gitlab.kitware.com/cmake/cmake/-/issues/20812
          /INCREMENTAL:NO
      )
    else()
      add_compile_options(
          -fno-omit-frame-pointer
          # Undefine NDEBUG, which is explicitly defined by the RelWithDebInfo CMake configuration, so that asserts are
          # kept.
          -UNDEBUG
      )
    endif()
  endif()

  if(CIFUZZ_ENGINE STREQUAL libfuzzer)
    if(MSVC)
      add_compile_options(/fsanitize=fuzzer)
    else()
      add_compile_options(-fsanitize=fuzzer)
    endif()
  elseif(CIFUZZ_ENGINE STREQUAL coverage)
    if(MSVC)
      message(FATAL_ERROR "CIFuzz: MSVC does not support coverage builds yet")
    else()
      add_compile_options(-fprofile-instr-generate -fcoverage-mapping)
      add_link_options(-fprofile-instr-generate -fcoverage-mapping)
    endif()
  endif()

  foreach(sanitizer IN LISTS CIFUZZ_SANITIZERS)
    if(sanitizer STREQUAL address)
      if(MSVC)
        # stack-use-after-scope instrumentation is enabled by default.
        # https://docs.microsoft.com/en-us/cpp/sanitizers/asan?view=msvc-170#differences
        add_compile_options(/fsanitize=address)
        # MSVC automatically signals to the linker that ASan should be linked.
        # https://docs.microsoft.com/en-us/cpp/build/reference/inferasanlibs?view=msvc-170
      else()
        add_compile_options(
            -fsanitize=address
            -fsanitize-recover=address
            -fsanitize-address-use-after-scope
        )
        add_link_options(-fsanitize=address)
      endif()
    elseif(sanitizer STREQUAL undefined)
      if(MSVC)
        message(FATAL_ERROR "CIFuzz: MSVC does not support UndefinedBehaviorSanitizer yet")
      else()
        add_compile_options(-fsanitize=undefined)
        add_link_options(-fsanitize=undefined)
        if (CMAKE_CXX_COMPILER_ID STREQUAL Clang)
          # To avoid issues with clang (not clang++) and UBSan, see
          # https://github.com/bazelbuild/bazel/issues/11122#issuecomment-896613570
          add_link_options(-fsanitize-link-c++-runtime)
        endif()
      endif()
    else()
      message(FATAL_ERROR "CIFuzz: Unsupported value in CIFUZZ_SANITIZERS: ${sanitizer}")
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

  if(CIFUZZ_USE_DEPRECATED_MACROS)
    # The old fuzz macro header is injected via the compile command line. It does not live under the include directory
    # so that is not offered to fuzz tests using the new macros via include path IDE completions.
    set(_fuzz_macro_header "$<SHELL_PATH:${CIFUZZ_INCLUDE_DIR}/../legacy/fuzz_macro.h>")
    if(MSVC)
      target_compile_options("${name}" PRIVATE /FI"${_fuzz_macro_header}")
    else()
      target_compile_options("${name}" PRIVATE "-include${_fuzz_macro_header}")
    endif()
  else()
    target_include_directories("${name}" SYSTEM PRIVATE "${CIFUZZ_INCLUDE_DIR}")
    # This macro is consumed by cifuzz.h.
    target_compile_definitions("${name}" PRIVATE CIFUZZ_TEST_NAME="${name}")
  endif()

  if(CIFUZZ_ENGINE STREQUAL replayer OR CIFUZZ_ENGINE STREQUAL coverage)
    target_link_libraries("${name}" PRIVATE cifuzz_internal_replayer)
  elseif(CIFUZZ_ENGINE STREQUAL libfuzzer)
    if(MSVC)
      # MSVC already marks its compilation outputs as requiring a link against libFuzzer and thus link.exe doesn't
      # offer the equivalent of `-fsanitize=fuzzer`.
    elseif(CMAKE_CXX_COMPILER_ID STREQUAL "Clang" OR ((NOT "CXX" IN_LIST LANGUAGES) AND (CMAKE_C_COMPILER_ID STREQUAL "Clang")))
      target_link_options("${name}" PRIVATE -fsanitize=fuzzer)
    else()
      message(FATAL_ERROR "CIFuzz: ${CMAKE_CXX_COMPILER_ID} compiler is not supported with the libfuzzer engine")
    endif()
  else()
    message(FATAL_ERROR "CIFuzz: Unsupported value for CIFUZZ_ENGINE: ${CIFUZZ_ENGINE}")
  endif()

  set(_seed_corpus_suffix _seed_corpus)
  set(_source_seed_corpus "${CMAKE_CURRENT_SOURCE_DIR}/${name}${_seed_corpus_suffix}")
  # Convert path separators to '\' (Windows only) and escape all backslashes for a C string literal.
  # In the regex strings below, one level of escaping is for the CMake string and another one to get a literal backslash
  # in a regex.
  if(WIN32)
    string(REGEX REPLACE "/" "\\\\" _source_seed_corpus "${_source_seed_corpus}")
  endif()
  string(REGEX REPLACE "\\\\" "\\\\\\\\" _source_seed_corpus "${_source_seed_corpus}")
  # Compile the path to the seed corpus, which lives under the source root, into the fuzz test binary as it is built
  # out-of-tree. An alternative could be to symlink the seed corpus to a well-known location next to the binary, but
  # symlinks are not always available on Windows (junctions exist, but may cause issues with tools that are unaware of
  # them and are not easy to deal with using just POSIX functions).
  target_compile_definitions("${name}" PRIVATE CIFUZZ_SEED_CORPUS="${_source_seed_corpus}")

  # Collect a mapping from CMake target names to information required by cifuzz. Currently, this includes the path of
  # the fuzz test executable as well as of its seed corpus.
  # We don't use add_custom_command here as we want the mapping to exist already after the configure step, not only
  # after the build step - this way, it is comparatively cheap to update the mapping since the actual build tool doesn't
  # have to run. IDEs may even refresh the metadata automatically for us.
  # Note: Removed and renamed targets leave behind their entry in this mapping. Since these files are cheap to
  #       regenerate, cifuzz can just delete the entire .cifuzz directory before each build (see enable_fuzz_testing).
  set(_executable_info_file "${CMAKE_BINARY_DIR}/$<CONFIG>/.cifuzz/fuzz_tests/${name}/executable")
  file(GENERATE
       OUTPUT "$<SHELL_PATH:${_executable_info_file}>"
       CONTENT $<TARGET_FILE:${name}>)
  set(_seed_corpus_info_file "${CMAKE_BINARY_DIR}/$<CONFIG>/.cifuzz/fuzz_tests/${name}/seed_corpus")
  file(GENERATE
       OUTPUT "$<SHELL_PATH:${_seed_corpus_info_file}>"
       CONTENT "${_source_seed_corpus}")

  add_test(NAME "${name}_regression_test" COMMAND "${name}")

  # Define an install component cifuzz_internal_deps_${name} that, when "installed", prints the full paths of the
  # transitive runtime dependencies, including system libraries, of the fuzz target to stdout in the form:
  #
  # -- CIFUZZ RESOLVED /lib/x86_64-linux-gnu/libgcc_s.so.1
  # -- CIFUZZ RESOLVED /home/user/git/cifuzz/tools/cmake/testdata/build/src/utils/libhelper.so
  # -- CIFUZZ RESOLVED /lib/x86_64-linux-gnu/libstdc++.so.6
  #
  # If any library couldn't be resolved (unambiguously), it is reported with a leading UNRESOLVED or CONFLICTING.
  install(CODE "
    file(GET_RUNTIME_DEPENDENCIES
        RESOLVED_DEPENDENCIES_VAR _resolved_deps
        UNRESOLVED_DEPENDENCIES_VAR _unresolved_deps
        CONFLICTING_DEPENDENCIES_PREFIX _conflicting_deps
        EXECUTABLES \"$<TARGET_FILE:${name}>\"
    )

    foreach(_resolved_dep IN LISTS _resolved_deps)
        message(STATUS \"CIFUZZ RESOLVED \${_resolved_dep}\")
    endforeach()
    foreach(_unresolved_dep IN LISTS _unresolved_deps)
        message(STATUS \"CIFUZZ UNRESOLVED \${_unresolved_dep}\")
    endforeach()
    foreach(_conflicting_dep IN LISTS _conflicting_deps)
        message(STATUS \"CIFUZZ CONFLICTING \${_conflicting_dep}\")
    endforeach()
  " COMPONENT "cifuzz_internal_deps_${name}")
endfunction()
