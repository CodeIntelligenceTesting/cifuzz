#!/bin/bash

set -euo pipefail
set -x

SCRIPT_DIR=$(dirname "$(readlink -f "$0")")

print_usage() {
  cat <<EOF
usage: $0 REPO REFERENCE

Update the vendored version of the specified repo to the specified
git reference.

Options:
-h, --help
    Print usage message.
EOF
}

# Parse arguments
POSITIONAL=()
while [ "$#" -gt 0 ]; do
  case "$1" in
    -h | --help)
      print_usage
      exit 0
      ;;
    -*) # unknown option
      echo >&2 "unknown option: $1"
      print_usage
      exit 1
      ;;
    *)                   # positional
      POSITIONAL+=("$1") # save it in an array for later
      shift              # past argument
      ;;
  esac
done
set -- "${POSITIONAL[@]}" # restore positional parameters

if [ "$#" -lt 2 ]; then
  print_usage
  exit 1
fi

REPO="$1"
REF="$2"

TARGET_DIR=$(basename ${REPO})
rm -rf "${SCRIPT_DIR}/${TARGET_DIR}"
git clone --depth 1 --branch "${REF}" "${REPO}" ${TARGET_DIR}
rm -rf "${TARGET_DIR}/.git"
echo "${REF}" >"${TARGET_DIR}/ref"
