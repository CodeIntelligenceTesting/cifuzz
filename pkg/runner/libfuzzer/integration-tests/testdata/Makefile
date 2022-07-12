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
EXT :=
OUT_OPTION := -o 
INCLUDE_OPTION := -I 
NOLINK_OPTION := -c
OUT_OBJ_OPTION := -o
DISABLE_UBSAN := 0

# handle MSVC specific options
ifeq ($(OS),Windows_NT)
	CC = cl
	CXX = cl
	CFLAGS = /fsanitize=fuzzer \
	  /fsanitize=address \
		/Zi

	CXXFLAGS = /fsanitize=fuzzer \
	  /fsanitize=address \
		/Zi \
		/std:c++17

	EXT = .exe
	OUT_OPTION = /Fe
	INCLUDE_OPTION = /I
	NOLINK_OPTION = /c
	OUT_OBJ_OPTION = /Fo

	DISABLE_UBSAN = 1
endif

ALL_PREREQUISITES = clean prepare $(FUZZ_TARGETS)

ifeq ($(DISABLE_UBSAN), 0)
	 ALL_PREREQUISITES += $(FUZZ_TARGET_DIR)/trigger_ubsan_non_recoverable;
	 FUZZ_TARGETS = $(shell find $(FUZZ_TARGET_DIR) -name '*.c*' | cut -f 1 -d '.')
else
	 FUZZ_TARGETS = $(shell find $(FUZZ_TARGET_DIR) -name '*.c*' ! -name '*ubsan*' | cut -f 1 -d '.')
endif

clean:
	-rm -rf $(BUILD_DIR)/*

prepare:
	mkdir -p $(BUILD_DIR)

all: $(ALL_PREREQUISITES)

$(BUILD_DIR)/api.obj: $(SHARED_DIR)/api.cpp $(SHARED_DIR)/api.h
	${CXX} $< ${CXXFLAGS} $(NOLINK_OPTION) $(INCLUDE_OPTION)$(SHARED_DIR) $(OUT_OBJ_OPTION)$@ 
	@echo "--> done: $@"

$(FUZZ_TARGET_DIR)/do_stuff_fuzzer: OUT=$(BUILD_DIR)/$(notdir $@)$(EXT)
$(FUZZ_TARGET_DIR)/do_stuff_fuzzer: ${FUZZ_TARGET_DIR}/do_stuff_fuzzer.cpp $(BUILD_DIR)/api.obj
	$(CXX) $? $(CXXFLAGS) $(INCLUDE_OPTION)$(SHARED_DIR) $(OUT_OPTION)$(OUT)
	@echo "--> done: $(OUT)"

$(FUZZ_TARGET_DIR)/trigger_asan_with_file_accesses: OUT=$(BUILD_DIR)/$(notdir $@)$(EXT)
$(FUZZ_TARGET_DIR)/trigger_asan_with_file_accesses: ${FUZZ_TARGET_DIR}/trigger_asan_with_file_accesses.c $(BUILD_DIR)/api.obj
	$(CC) $? $(CFLAGS) $(INCLUDE_OPTION)$(SHARED_DIR) $(OUT_OPTION)$(OUT)
	@echo "--> done: $(OUT)"

$(FUZZ_TARGET_DIR)/trigger_ubsan_non_recoverable: OUT=$(BUILD_DIR)/$(notdir $@)$(EXT)
$(FUZZ_TARGET_DIR)/trigger_ubsan_non_recoverable: $(FUZZ_TARGET_DIR)/trigger_ubsan.cpp
	$(CXX) $? $(CXXFLAGS) -fno-sanitize-recover=undefined $(OUT_OPTION)$(OUT) 
	@echo "--> done: $(OUT)"

$(FUZZ_TARGET_DIR)/%: OUT=$(BUILD_DIR)/$(notdir $@)$(EXT)
$(FUZZ_TARGET_DIR)/%: $(FUZZ_TARGET_DIR)/%.c prepare
	$(CC) $< $(CFLAGS) $(OUT_OPTION)$(OUT)
	@echo "--> done: $(OUT)"

$(FUZZ_TARGET_DIR)/%: OUT=$(BUILD_DIR)/$(notdir $@)$(EXT)
$(FUZZ_TARGET_DIR)/%: $(FUZZ_TARGET_DIR)/%.cpp prepare
	$(CXX) $< $(CXXFLAGS) $(OUT_OPTION)$(OUT)
	@echo "--> done: $(OUT)"
