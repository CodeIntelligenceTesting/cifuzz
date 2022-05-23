CC = clang
CXX = clang++ 

# Keep this flags in sync with the flags used in the make integration
# see also `internal/cmd/run/run.go`
CFLAGS = -fsanitize=fuzzer,address,undefined \
		-fno-sanitize=function,vptr \
		-fsanitize-recover=address \
		-fsanitize-address-use-after-scope \
		-g

CXXFLAGS = -fsanitize=fuzzer,address,undefined \
		-fno-sanitize=function,vptr \
		-fsanitize-recover=address \
		-fsanitize-address-use-after-scope \
		-std=c++17 \
		-g

BUILD_DIR = build
SHARED_DIR = shared
FUZZ_TARGET_DIR = fuzz-targets
# get all c/cpp files and remove file extension
FUZZ_TARGETS = $(shell find $(FUZZ_TARGET_DIR) -name '*.c*' | cut -f 1 -d '.')

clean:
	-rm -rf $(BUILD_DIR)/*

prepare:
	-mkdir $(BUILD_DIR)

all: clean prepare $(FUZZ_TARGETS) trigger_ubsan_non_recoverable;

$(BUILD_DIR)/api.o: $(SHARED_DIR)/api.cpp $(SHARED_DIR)/api.h
	${CXX} ${CXXFLAGS} -c -I $(SHARED_DIR) -o $@ $< 

$(BUILD_DIR)/api.a: $(BUILD_DIR)/api.o
	llvm-ar rc $@ $<

$(FUZZ_TARGET_DIR)/do_stuff_fuzzer: ${FUZZ_TARGET_DIR}/do_stuff_fuzzer.cpp $(BUILD_DIR)/api.a
	$(CXX) $(CXXFLAGS) -I $(SHARED_DIR) -o $(BUILD_DIR)/$(notdir $@) $?

$(FUZZ_TARGET_DIR)/trigger_asan_with_file_accesses: ${FUZZ_TARGET_DIR}/trigger_asan_with_file_accesses.c $(BUILD_DIR)/api.a
	$(CC) $(CFLAGS) -I $(SHARED_DIR) -o $(BUILD_DIR)/$(notdir $@) $?
	cp files/example.conf $(BUILD_DIR)/

$(FUZZ_TARGET_DIR)/trigger_oom: $(FUZZ_TARGET_DIR)/trigger_oom.cpp
	$(CXX) $(CXXFLAGS) -Wno-unused-but-set-variable $< -o $(BUILD_DIR)/$(notdir $@)

trigger_ubsan_non_recoverable: $(FUZZ_TARGET_DIR)/trigger_ubsan.cpp
	$(CXX) $(CXXFLAGS) -fno-sanitize-recover=undefined $< -o $(BUILD_DIR)/$(notdir $@)

$(FUZZ_TARGET_DIR)/%: $(FUZZ_TARGET_DIR)/%.c
	$(CC) $(CFLAGS) $< -o $(BUILD_DIR)/$(notdir $@)

$(FUZZ_TARGET_DIR)/%: $(FUZZ_TARGET_DIR)/%.cpp
	$(CXX) $(CXXFLAGS) $< -o $(BUILD_DIR)/$(notdir $@)

