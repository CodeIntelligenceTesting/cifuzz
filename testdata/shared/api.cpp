// Copyright 2017 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");

// Implementation of "api".
#include "api.h"

#include <iostream>
#include <vector>

// Do some computations with 'str', return the result.
// This function contains a bug. Can you spot it?
size_t DoStuff(const std::string &str) {
  std::vector<int> Vec({0, 1, 2});
  size_t Idx = 0;
  if (str.size() > 5) Idx++;
  if (str.find("foo") != std::string::npos) Idx += 2;
  return Vec[Idx];
}

size_t DoStuffWithMyStruct(struct MyStruct m) {
  if (m.extra_value > 100 && m.data_size == 27) {
    std::string s(m.data, m.data_size);
    //    if (s.find("deadbeef") == 0) {
    *(char *)1 = 2;
    //    }
  }
  return 0;
}

void LeakMemory() {
  void *p = malloc(6);
  if (p) {
    *(char *)p = 'B';
  }
}
