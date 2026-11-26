# Func Store

[![Go Report Card](https://goreportcard.com/badge/github.com/dracory/funcstore)](https://goreportcard.com/report/github.com/dracory/funcstore)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/dracory/funcstore)](https://pkg.go.dev/github.com/dracory/funcstore)

Func Store is a Go package that provides persistent storage and execution for
serverless functions (FaaS). Store a function once, get an ID, invoke it
anywhere — in any supported runtime.

It is runtime-agnostic by design: runtimes are pluggable via the `Runtime`
interface rather than bolted on. Ships with three runtimes:

- **js** — JavaScript via [`dop251/goja`](https://github.com/dop251/goja)
- **go** — Go via [`traefik/yaegi`](https://github.com/traefik/yaegi) (Go interpreter)
- **php** — PHP via [`titpetric/phpscript`](https://github.com/titpetric/phpscript)

Backed by [`dracory/neat`](https://github.com/dracory/neat), so it works with
any supported database (SQLite, MySQL, Postgres, etc.) through a single
`*sql.DB`.

## Table of contents

1. [Features](#features)
2. [Installation](#installation)
3. [Quick start](#quick-start)
4. [Functions & versions](#functions--versions)
5. [Groups](#groups)
6. [Runtimes](#runtimes)
7. [Testing](#testing)
8. [License](#license)

## Features

- **Persistent function registry** — functions are stored in the database with
  alias/alias-path lookup, metadata, timeout and memory-limit settings.
- **Pluggable runtimes** — implement `Runtime`/`Program` to add new languages.
- **Source versioning** — every save snapshots the source; `FunctionRestore`
  rolls back to any previous version.
- **Groups** — hierarchical organisation with denormalised paths
  (`GroupCreate` resolves `Path` automatically).
- **Invoke by ID, alias path, or alias** — `FunctionInvoke` tries each in turn.
- **Compile-checked on save** — `FunctionCreate` fails fast if the source does
  not compile under the registered runtime.
- **Auto-migration** — optional schema bootstrap via `AutomigrateEnabled`.

## Installation

```bash
go get github.com/dracory/funcstore
```

## Quick start

```go
package main

import (
    "context"
    "database/sql"
    _ "modernc.org/sqlite" // or your preferred driver

    "github.com/dracory/funcstore"
)

func main() {
    db, err := sql.Open("sqlite", "file:faas.db")
    if err != nil {
        panic(err)
    }

    store, err := funcstore.NewStore(funcstore.StoreOptions{
        DB:                 db,
        Driver:             "sqlite",
        Runtimes:           []funcstore.Runtime{funcstore.NewGojaRuntime()},
        AutomigrateEnabled: true,
    })
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    err = store.FunctionCreate(ctx, funcstore.NewFunction().
        SetRuntime("js").
        SetSource("return input * 2").
        SetAlias("double"))
    if err != nil {
        panic(err)
    }

    out, err := store.FunctionInvoke(ctx, "double", 21) // -> 42
    if err != nil {
        panic(err)
    }
    println(out.(int64))
}
```

## Functions & versions

`FunctionInterface` exposes fluent setters/getters (`SetRuntime`, `SetSource`,
`SetAlias`, `SetTimeoutMs`, `SetMetadata`, ...). On every update the current
source is snapshotted into `funcstore_function_versions`; manage them with
`FunctionVersionList`, `FunctionVersionGet`, `FunctionVersionDelete`, and
`FunctionRestore` (which first snapshots the live source so the rollback itself
is recoverable).

## Groups

Groups form a tree (`parent_id` + denormalised `path`). `GroupCreate` accepts a
group whose `Parent` is either an ID or a path, and refuses to delete
non-empty groups unless `DeleteOpts.Cascade` is set.

## Runtimes

A `Runtime` compiles source into a `Program`; `Program.Run(input)` executes it
with a per-invocation timeout. Register the ones you need via
`StoreOptions.Runtimes`:

```go
Runtimes: []funcstore.Runtime{
    funcstore.NewGojaRuntime(),
    funcstore.NewYaegiRuntime(),
    funcstore.NewPhpscriptRuntime(),
}
```

The PHP runtime sandboxes filesystem access to `data/faas/files` relative to
the working directory.

## Testing

```bash
go test ./...
```

Tests run against in-memory SQLite via `modernc.org/sqlite` — no external
services required.

## License

This project is licensed under the MIT License. You can find a copy of the
license at [LICENSE](LICENSE).
