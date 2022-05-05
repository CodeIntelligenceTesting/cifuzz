#ifndef CIFUZZ_TOOLS_CMAKE_TESTDATA_SRC_PARSER_PARSER_H
#define CIFUZZ_TOOLS_CMAKE_TESTDATA_SRC_PARSER_PARSER_H

#include <string>

#ifdef _WIN32
__declspec(dllexport)
#endif
int parse(const std::string &input);

#endif // CIFUZZ_TOOLS_CMAKE_TESTDATA_SRC_PARSER_PARSER_H
