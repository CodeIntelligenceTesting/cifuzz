#!/bin/bash

set -eu

CIFUZZ_INSTALL_ROOT=$(mktemp -d)
mkdir -p "${CIFUZZ_INSTALL_ROOT}/bin"
mkdir -p "${CIFUZZ_INSTALL_ROOT}/lib"

function install() {
  # Install minijail0
  pushd third-party/minijail
  make 'CC_BINARY(minijail0)'
  make 'CC_LIBRARY(libminijailpreload.so)'
  cp minijail0 "${CIFUZZ_INSTALL_ROOT}/bin"
  cp libminijailpreload.so "${CIFUZZ_INSTALL_ROOT}/lib"
  popd

  # Install process_wrapper
  "${CC:-clang}" -o "${CIFUZZ_INSTALL_ROOT}/lib/process_wrapper" pkg/minijail/process_wrapper/src/process_wrapper.c

}

# We want the only output on stdout of this script to be the
# CIFUZZ_INSTALL_ROOT variable, to allow the caller to easily set that
# in their environment. Therefore, we redirect all output from the
# installation to stderr via 1>&2.
install 1>&2

echo "CIFUZZ_INSTALL_ROOT=${CIFUZZ_INSTALL_ROOT}"
