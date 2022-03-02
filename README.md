# negroni-zerolog

[![PkgGoDev](https://pkg.go.dev/badge/github.com/samvdb/negroni-zerolog)](https://pkg.go.dev/github.com/samvdb/negroni-zerolog)
[![Main Workflow Status](https://github.com/samvdb/negroni-zerolog/workflows/main/badge.svg)](https://github.com/samvdb/negroni-zerolog/workflows/main)


Adaptation of [meatballhat/negroni-logrus](https://github.com/meatballhat/negroni-logrus).

logrus middleware for negroni

## Usage

Create a new middleware with `NewMiddleware`

If you want to reuse an already initialized `zerolog.Logger`, you should be using
`NewMiddlewareFromLogger` 