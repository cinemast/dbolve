[![CircleCI](https://circleci.com/gh/cinemast/dbolve.svg?style=svg)](https://circleci.com/gh/cinemast/dbolve)
[![codecov](https://codecov.io/gh/cinemast/dbolve/branch/master/graph/badge.svg)](https://codecov.io/gh/cinemast/dbolve)
[![GoDoc](https://godoc.org/github.com/cinemast/dbolve/go?status.svg)](https://godoc.org/github.com/cinemast/dbolve)
[![Go Report Card](https://goreportcard.com/badge/github.com/cinemast/dbolve)](https://goreportcard.com/report/github.com/cinemast/dbolve)

# dbolve

Very simple code only migration library for go

## Features
- Very simple and readable code (< 200 lines of code)
- Easy to use interface
- Transaction safety for each migration
- Verifies that already applied transactions haven't changed

## Usage
```
go get -u github.com/cinemast/dbolve
```

### Quickstart

[examples/main.go](examples/main.go)

## Motivation

Heavily inspired by [lopezator/migrator](https://github.com/lopezator/migrator).

I was missing two features:
  - Allow to list already applied and pending migrations
  - Verification that already applied migrations match the current migration code
