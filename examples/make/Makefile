CXXFLAGS += -std=c++11

all: main

clean:
	rm -f *.o *.a *.so a.out

libapi.so: api.cpp api.h
	${CXX} ${CXXFLAGS} -shared -fpic -o libapi.so $<

api.o: api.cpp api.h
	${CXX} ${CXXFLAGS} -fpic $< -c

api.a: api.o
	ar rv api.a api.o

main: api.o
	${CXX} ${CXXFLAGS} api.o main.cpp

my_fuzz_test%: libapi.so
	@echo "Building $@"
	${CXX} ${CXXFLAGS} ${FUZZ_TEST_CFLAGS} ${FUZZ_TEST_LDFLAGS} -o $@ $@.cpp -Wl,-rpath '-Wl,$$ORIGIN' -L. -lapi
