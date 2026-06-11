# di

[![Go Reference](https://pkg.go.dev/badge/github.com/shanjunmei/di.svg)](https://pkg.go.dev/github.com/shanjunmei/di)
[![Go Report Card](https://goreportcard.com/badge/github.com/shanjunmei/di)](https://goreportcard.com/report/github.com/shanjunmei/di)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**di** is a minimalistic dependency injection container for Go.  
It has **no code generation**, **no external dependencies** (only the standard library), and provides a simple API: `Provide`, `Invoke`, and `Module`.  
It follows the philosophy: *keep it simple, stay idiomatic*.

## Features

- ✅ **Runtime only** – uses reflection, no code generation.
- ✅ **Singleton by default** – each constructor is called once, result is cached.
- ✅ **Error support** – constructors may return `(T, error)`; `Invoke` functions may return `error`.
- ✅ **Modular** – combine multiple `Provide`/`Invoke` with `Module`.
- ✅ **Zero dependencies** – only Go standard library.
- ✅ **Tiny binary overhead** – only a few tens of KB.

## Installation

    go get github.com/shanjunmei/di

Requires Go 1.22+ (for `range over int`; if you need older Go, replace `for i := range n` with traditional `for i := 0; i < n; i++`).

## Quick Start

    package main

    import (
        "context"
        "fmt"
        "log"

        "github.com/shanjunmei/di"
    )

    type Config struct {
        Addr string
    }

    func NewConfig() *Config {
        return &Config{Addr: ":8080"}
    }

    type Server struct {
        cfg *Config
    }

    func NewServer(cfg *Config) *Server {
        return &Server{cfg: cfg}
    }

    func (s *Server) Run() error {
        fmt.Printf("Server listening on %s\n", s.cfg.Addr)
        return nil
    }

    func main() {
        app := di.New(
            di.Provide(NewConfig),
            di.Provide(NewServer),
            di.Invoke(func(srv *Server) error {
                return srv.Run()
            }),
        )

        if err := app.Run(context.Background()); err != nil {
            log.Fatal(err)
        }
    }

## Module Composition

    // user/module.go
    func Module() di.Option {
        return di.Module(
            di.Provide(repository.New),
            di.Provide(service.New),
            di.Provide(handler.New),
            di.Invoke(RegisterRoutes),
        )
    }

    // main.go
    app := di.New(
        user.Module(),
        order.Module(),
        di.Invoke(startServer),
    )

> For a complete runnable example, see [examples/main.go](./examples/main.go).

## Comparison with Mainstream DI Frameworks

| Feature                     | di            | Google Wire           | Uber Fx                |
| --------------------------- | ------------- | --------------------- | ---------------------- |
| Code generation             | ❌ No         | ✅ `wire gen` required | ❌ No                  |
| Runtime reflection          | ✅ Yes        | ❌ No (compile-time)   | ✅ Yes                 |
| Startup performance         | Medium        | Very fast             | Slow                   |
| Compile‑time type safety    | ❌ No         | ✅ Yes                | ❌ No                  |
| Lifecycle hooks             | ❌ None       | ❌ None               | ✅ `OnStart`/`OnStop`  |
| Dependency order guarantee  | Registration order | Topological sort | Sort + lifecycle       |
| External dependencies       | 0             | 0 (generated code)    | Multiple               |
| Learning curve              | Very low      | Medium                | High                   |
| Module composition          | `Module`      | `wire.NewSet`         | `fx.Module`            |
| Final binary size           | Tiny (~50KB)  | Tiny                  | Larger                 |
| Best suited for             | Small/medium projects, quick prototypes | Large projects, startup‑critical | Large projects, complex lifecycle |

### Which One to Choose?

- **You want simplicity, zero deps, fast startup** → `di`
- **You need compile‑time safety and max startup performance** → Wire (original archived, use community fork)
- **You need lifecycle management (start/stop hooks, plugin collection)** → Fx

## Why Not Code Generation?

- Avoid introducing extra `go generate` steps.
- Runtime flexibility – you can conditionally register components (e.g., based on config).
- Small footprint – ideal for embedding in other projects.

## License

MIT
