add_library(helper SHARED helper.cpp)
target_sources(helper PUBLIC
               helper.h)
target_link_libraries(helper PRIVATE secrets)
target_include_directories(helper PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})
