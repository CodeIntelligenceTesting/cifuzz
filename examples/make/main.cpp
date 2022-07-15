#include "api.h"

int main(int argc, char *argv[]) {
  for (int i = 1; i < argc; i++) {
    Read(argv[i]);

    std::string s(argv[i]);
    DoStuff(s);
  }
  return 0;
}
