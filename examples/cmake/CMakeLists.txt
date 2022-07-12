cmake_minimum_required(VERSION 3.19)
project(cmake_example)

enable_testing()

find_package(cifuzz)
enable_fuzz_testing()

add_subdirectory(src)

add_executable(${PROJECT_NAME} main.cpp )
target_link_libraries(${PROJECT_NAME} PRIVATE exploreMe)

add_fuzz_test(my_fuzz_test my_fuzz_test.cpp)
target_link_libraries(my_fuzz_test PRIVATE exploreMe)


