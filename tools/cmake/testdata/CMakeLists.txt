cmake_minimum_required(VERSION 3.19)

project(CmakeTest)

enable_testing()

find_package(cifuzz)
enable_fuzz_testing()

add_subdirectory(src/parser)
