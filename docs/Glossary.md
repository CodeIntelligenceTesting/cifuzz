# Glossary
To make sure we are talking about the same things we rely on the 
following definitions:

## Fuzzer aka Fuzzing Engine
Is a piece of software that generates input to feed into the system 
under test (SUT) via the fuzz targets. A coverage guided fuzzer will 
also collect information about the covered code during the execution 
and will try to improve coverage by adjusting the inputs accordingly.

## Fuzz Target
Usually a function that takes an array of bytes from the fuzzer and 
calls functions from the system under test.

## Fuzzing 
Fuzzing describes the act of running the fuzzer against a fuzz target.

## Corpus
A set of useful test inputs

### Seed Corpus
A small corpus usually provided by the user. Findings can be added to 
this corpus for use in regression tests.

### Generated Corpus
The generated corpus contains interesting inputs found by the fuzzer 
while running the fuzz test.

## Replayer
A replayer runs inputs from the seed corpus against the fuzz target 
without generating new input. It can be used for debugging and/or 
regression testing.

## Sanitizer
Santizers use code instrumentation to detect bugs during the execution 
of an application. 

## Finding
A finding describes a bug or vulnerability found by the fuzzer and 
includes the input that causes it.

## Metrics
While using **cifuzz** you will get in touch with metrics descriping 
the progress or outcome of a fuzzing run. Some of them might have 
different meanings depending on the context, previous knowledge or 
might even be used differently by the various fuzzing engines. 
To avoid confusion we provide the following definitions:

### Features

### Branches

### Coverage

