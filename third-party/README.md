# Vendoring

We currently only vendor code which we want to build and ship ourselves.

## Jazzer

The current Jazzer version can be found in `jazzer/ref`.

To update the vendored Jazzer code to a specific git reference `$REF`:

```bash
./update.sh https://github.com/CodeIntelligenceTesting/jazzer "${REF}"
```

## minijail

The current minijail version can be found in `minijail/ref`.

To update the vendored minijail code to a specific git reference `$REF`:

```bash
./update.sh https://android.googlesource.com/platform/external/minijail "${REF}"
```
