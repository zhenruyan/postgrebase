(EXP) tygoja
[![GoDoc](https://godoc.org/github.com/pocketbase/tygoja?status.svg)](https://pkg.go.dev/github.com/pocketbase/tygoja)
======================================================================

**tygoja** is a small helper library for generating TypeScript declarations from Go code.

The generated typings are intended to be used as import helpers to provide [ambient TypeScript declarations](https://www.typescriptlang.org/docs/handbook/declaration-files/introduction.html) (aka. `.d.ts`) for [goja](https://github.com/dop251/goja) bindings.

> **⚠️ Don't use it directly in production! It is not versioned and may change without notice.**
>
> **It was created to semi-automate the documentation of the goja integration for PocketBase.**
>
> **Use it only as a learning reference or as a non-critical step in your dev pipeline.**

**tygoja** is a heavily modified fork of [tygo](https://github.com/gzuidhof/tygo) and extends its scope with:

- custom field and method names formatters
- types for interfaces (exported and unexported)
- types for exported interface methods
- types for exported struct methods
- types for exported package level functions (_must enable `PackageConfig.WithPackageFunctions`_)
- inheritance declarations for embeded structs (_embedded pointers are treated as values_)
- autoloading all unmapped argument and return types (_when possible_)
- applying the same [goja's rules](https://pkg.go.dev/github.com/dop251/goja#hdr-Nil) when resolving the return types of exported function and methods
- combining multiple packages typings in a single output
- generating all declarations in namespaces with the packages name (both unmapped and mapped)
- preserving comment block new lines
- converting Go comment code blocks to Markdown code blocks
- and others...


## Known issues and limitations

- Multiple versions of the same package may have unexpected declaration as all versions will be under the same namespace
- For easier generation, it relies on TypeScript declarations merging, meaning that the generated types may not be very compact

## Example

```go
package main

import (
    "log"
    "os"

    "github.com/pocketbase/tygoja"
)

func main() {
    gen := tygoja.New(tygoja.Config{
        Packages: map[string][]string{
            "github.com/pocketbase/tygoja/test/a": {"*"},
            "github.com/pocketbase/tygoja/test/b": {"*"},
            "github.com/pocketbase/tygoja/test/c": {"Example2", "Handler"},
        },
        Heading:              `declare var $app: c.Handler;`,
        WithPackageFunctions: true,
    })

    result, err := gen.Generate()
    if err != nil {
        log.Fatal(err)
    }

    if err := os.WriteFile("./types.d.ts", []byte(result), 0644); err != nil {
        log.Fatal(err)
    }
}
```

You can also combine it with [typedoc](https://typedoc.org/) to create HTML/JSON docs from the generated declaration(s).

See the package `/test` directory for example output.
