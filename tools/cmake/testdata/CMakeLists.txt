cmake_minimum_required(VERSION 3.21)

project(CmakeTest)

enable_testing()

find_package(cifuzz)
enable_fuzz_testing()

add_subdirectory(src/parser)
add_subdirectory(src/utils)
