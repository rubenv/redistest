# redistest

> Go library to spawn single-use Redis servers for unit testing

[![Build Status](https://github.com/rubenv/redistest/workflows/Test/badge.svg)](https://github.com/rubenv/redistest/actions) [![Build Status](https://travis-ci.org/rubenv/redistest.svg?branch=master)](https://travis-ci.org/rubenv/redistest) [![GoDoc](https://godoc.org/github.com/rubenv/redistest?status.png)](https://godoc.org/github.com/rubenv/redistest)

Spawns a Redis server. Ideal for unit tests where you want a clean instance
each time. Then clean up afterwards.

Features:

* Starts a clean isolated Redis database
* Tested on Fedora, Ubuntu and Alpine
* Optimized for in-memory execution, to speed up unit tests
* Less than 0.1 second startup / initialization time

## Usage

In your unit test:
```go
red, err := redistest.Start()
defer red.Stop()

// Do something with red.Pool (which is a *redis.Pool)
```

## License

This library is distributed under the [MIT](LICENSE) license.
