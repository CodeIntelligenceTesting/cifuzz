#ifndef CMAKETEST_TOOLS_CMAKE_TESTDATA_SRC_UTILS_HELPER_H
#define CMAKETEST_TOOLS_CMAKE_TESTDATA_SRC_UTILS_HELPER_H

#include <string>

#ifdef _WIN32
__declspec(dllexport)
#endif
int do_something_weird(const std::string &input);

#endif // CMAKETEST_TOOLS_CMAKE_TESTDATA_SRC_UTILS_HELPER_H
