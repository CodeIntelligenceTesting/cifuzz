// Copyright 2017 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");

// Implementation of "api".
#include "api.h"

#include <iostream>
#include <string.h>
#include <vector>

// Do some computations with 'str', return the result.
// This function contains a bug. Can you spot it?
size_t DoStuff(const std::string &str) {
  std::vector<int> Vec({0, 1, 2});
  size_t Idx = 0;
  if (str.size() > 5)
    Idx++;
  if (str.find("foo") != std::string::npos)
    Idx += 2;
  return Vec[Idx];
}

void Read(char *src) {
  char dst[10];
  memcpy(src, dst, strlen(src));
}