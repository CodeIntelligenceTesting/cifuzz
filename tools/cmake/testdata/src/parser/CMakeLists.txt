add_library(secrets secrets.cpp)
target_sources(secrets PUBLIC
               secrets.h)
set_property(TARGET secrets PROPERTY POSITION_INDEPENDENT_CODE ON)
target_include_directories(secrets PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})

add_library(parser parser.cpp)
target_sources(parser PUBLIC
               parser.h)
target_link_libraries(parser PRIVATE helper)
target_include_directories(parser PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})

if(CIFUZZ_USE_DEPRECATED_MACROS)
  add_fuzz_test(legacy_fuzz_test legacy_fuzz_test.cpp)
else()
  add_fuzz_test(parser_fuzz_test parser_fuzz_test.cpp)
  target_link_libraries(parser_fuzz_test PRIVATE parser)
  if(WIN32)
    # Copy all DLLs right next to the binary to account for the lack of R(UN)PATH on Windows.
    add_custom_command(TARGET parser_fuzz_test POST_BUILD
                       COMMAND ${CMAKE_COMMAND} -E copy $<TARGET_RUNTIME_DLLS:parser_fuzz_test> $<TARGET_FILE_DIR:parser_fuzz_test>
                       COMMAND_EXPAND_LISTS)
  endif()

  add_fuzz_test(no_seed_corpus_fuzz_test crash_always_fuzz_target.cpp)

  add_fuzz_test(c_fuzz_test c_fuzz_test.c)
endif()
