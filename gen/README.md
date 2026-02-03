# gen

This directory contains the code generation entrypoint for model/store code.

Usage:

```
go run gen/gen.go
```

Notes:
- The generator calls `store.Generate()` from `model/store`.
- The blank import in `gen.go` ensures store registrations are included.
