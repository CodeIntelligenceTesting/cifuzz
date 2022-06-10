#ifndef CORE_FUZZING_FUZZING_H
#define CORE_FUZZING_FUZZING_H

#define FUZZ_INIT_WITH_ARGS(c, v) LLVMFuzzerInitialize(c, v)

// since the libfuzzer documentation advices against using LLVMFuzzerInitialize
// we define this alternate macro that globally calls a function without
// arguments, ignoring the return code
#define FUZZ_INIT()                   \
  globalFuzzerInit();                 \
  int _init_res = globalFuzzerInit(); \
  int globalFuzzerInit()

#define FUZZ(d, s) LLVMFuzzerTestOneInput(d, s)

#endif  // CORE_FUZZING_FUZZING_H
