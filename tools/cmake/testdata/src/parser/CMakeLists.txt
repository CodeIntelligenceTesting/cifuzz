add_library(secrets secrets.cpp)
target_sources(secrets PUBLIC
               secrets.h)
set_property(TARGET secrets PROPERTY POSITION_INDEPENDENT_CODE ON)

add_library(parser SHARED parser.cpp)
target_sources(parser PUBLIC
               parser.h)
target_include_directories(parser PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})
target_link_libraries(parser PRIVATE secrets)

add_fuzz_test(parser_fuzz_test parser_fuzz_test.cpp)
target_link_libraries(parser_fuzz_test PRIVATE parser)

add_fuzz_test(no_seed_corpus_fuzz_test crash_always_fuzz_target.cpp)
