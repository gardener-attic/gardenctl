# Slices
[![PkgGoDev](https://pkg.go.dev/badge/github.com/srfrog/slices)](https://pkg.go.dev/github.com/srfrog/slices)
[![Go Report Card](https://goreportcard.com/badge/github.com/srfrog/slices?svg=1)](https://goreportcard.com/report/github.com/srfrog/slices)
[![codecov](https://codecov.io/gh/srfrog/slices/branch/master/graph/badge.svg?token=IDUWTIYYZQ)](https://codecov.io/gh/srfrog/slices)
![Build Status](https://github.com/srfrog/slices/workflows/Go/badge.svg)

*Functions that operate on slices. Similar to functions from `package strings` or `package bytes` that have been adapted to work with slices.*

## Features

- [x] Using a thin layer of idiomatic Go; correctness over performance.
- [x] Provide most basic slice operations: index, trim, filter, map
- [x] Some PHP favorites like: pop, push, shift, unshift, shuffle, etc...
- [x] Non-destructive returns (won't alter original slice), except for explicit tasks.

## Quick Start

Install using "go get":

```bash
go get github.com/srfrog/slices
```

Then import from your source:

```
import "github.com/srfrog/slices"
```

View [example_test.go][1] for examples of basic usage and features.

## Documentation

The full code documentation is located at GoDoc:

[http://godoc.org/github.com/srfrog/slices](http://godoc.org/github.com/srfrog/slices)

## Usage

This is a en example showing basic usage.

```go
package main

import(
   "fmt"

   "github.com/srfrog/slices"
)

func main() {
	str := `Don't communicate by sharing memory - share memory by communicating`

	// Split string by spaces into a slice.
	slc := strings.Split(str, " ")

	// Count the number of "memory" strings in slc.
	memories := slices.Count(slc, "memory")
	fmt.Println("Memories:", memories)

	// Split slice into two parts.
	parts := slices.Split(slc, "-")
	fmt.Println("Split:", parts, len(parts))

	// Compare second parts slice with original slc.
	diff := slices.Diff(slc, parts[1])
	fmt.Println("Diff:", diff)

	// Chunk the slice
	chunks := slices.Chunk(parts[0], 1)
	fmt.Println("Chunk:", chunks)

	// Merge the parts
	merge := slices.Merge(chunks...)
	fmt.Println("Merge:", merge)
}
```

[1]: https://github.com/srfrog/slices/blob/master/example_test.go
