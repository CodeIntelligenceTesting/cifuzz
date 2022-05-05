cmake_minimum_required(VERSION 3.21)

project(CmakeTest)

enable_testing()

list(APPEND CMAKE_PREFIX_PATH "${PROJECT_SOURCE_DIR}/..")
find_package(CIFuzz)
enable_fuzz_testing()

add_subdirectory(src/parser)
