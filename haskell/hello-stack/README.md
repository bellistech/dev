# hello-stack

Minimal Stack/Cabal project that prints "Hello, Stack!".

## Run with Stack (stays in repo)
```bash
STACK_ROOT=$PWD/.stack stack --system-ghc --no-install-ghc run
```

## Run with cabal (stays in repo)
```bash
CABAL_DIR=$PWD/.cabal CABAL_CONFIG=$PWD/.cabal/config cabal v2-run hello-stack
```

If network access to Hackage/Stackage is blocked, Stack/Cabal will try to download the package index and fail. Allow network briefly or mirror an index locally, then rerun the commands above. Caches stay inside this directory because of `STACK_ROOT` and `CABAL_DIR`.
