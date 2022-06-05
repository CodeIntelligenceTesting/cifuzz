# Git Hooks

This directory contains git hooks which can be useful when working on
this repository.

You can install them by running the following:

```bash
cd "$(git rev-parse --show-toplevel)/.git"
if [ -e hooks ]; then
  mv hooks hooks.orig
fi
ln -sf ../githooks hooks
```
