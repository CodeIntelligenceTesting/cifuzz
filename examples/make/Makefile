CXXFLAGS += -std=c++11

all: main

clean:
	rm -f *.o *.a *.so a.out

api.o: api.cpp api.h
	${CXX} ${CXXFLAGS} -fpic $< -c

api.a: api.o
	ar rv api.a api.o

main: api.o
	${CXX} ${CXXFLAGS} api.o main.cpp

my_fuzz_test%: api.o
	@echo "Building $@"
	${CXX} ${CXXFLAGS} ${FUZZ_TEST_CFLAGS} ${FUZZ_TEST_LDFLAGS} -o $@ $@.cpp api.o
