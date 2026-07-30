# Vedio

Vedio is a lightweight dependency injection container for Go. It helps you register implementations for interfaces and resolve them with configurable lifecycles such as singleton, transient, and scoped instances.

## Features

- Register interface-to-implementation mappings
- Support for singleton, transient, and scoped lifecycles
- Automatic object creation through an `Init` method
- Dependency injection for constructor-style initialization
- Optional custom generators and scope-based cleanup

## Installation

```bash
go get github.com/vedadiyan/vedio
```

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/vedadiyan/vedio"
)

type Greeter interface {
    Greet() string
}

type EnglishGreeter struct{}

func (g *EnglishGreeter) Init() error {
    return nil
}

func (g *EnglishGreeter) Greet() string {
    return "Hello from Vedio"
}

func main() {
    _ = vedio.RegisterFor[Greeter, EnglishGreeter]()

    greeter, err := vedio.Resolve[Greeter]()
    if err != nil {
        panic(err)
    }

    fmt.Println(greeter.Greet())
}
```

## Lifecycle Options

By default, registrations use the singleton lifecycle.

- `SINGLETON`: one instance is created and reused
- `TRANSIENT`: a new instance is created on every resolution
- `SCOPED`: one instance is reused within a specific scope

Example:

```go
_ = vedio.RegisterFor[Greeter, EnglishGreeter](
    vedio.LifeCycleOpt(vedio.TRANSIENT),
)
```

Scoped resolution:

```go
scope := vedio.NewScope()
defer scope.Close()

instance, err := vedio.Resolve[Greeter](vedio.ScopeOpt(scope))
```

## Testing

Run the test suite with:

```bash
go test ./...
```

## Project Structure

- `vedio.go`: core container implementation
- `vedio_test.go`: unit tests for container behavior
- `vedio_backbox_test.go`: integration-style behavior tests
