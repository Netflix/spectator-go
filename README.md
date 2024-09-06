[![Snapshot](https://github.com/Netflix/spectator-go/actions/workflows/snapshot.yml/badge.svg)](https://github.com/Netflix/spectator-go/actions/workflows/snapshot.yml)
[![Release](https://github.com/Netflix/spectator-go/actions/workflows/release.yml/badge.svg)](https://github.com/Netflix/spectator-go/actions/workflows/release.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Netflix/spectator-go.svg)](https://pkg.go.dev/github.com/Netflix/spectator-go)

## Spectator-go

Go thin-client metrics library for use with [Atlas] and [SpectatorD].

See the [Atlas Documentation] site for more details on `spectator-go`.

[Atlas]: https://netflix.github.io/atlas-docs/overview/
[SpectatorD]: https://netflix.github.io/atlas-docs/spectator/agent/usage/
[Atlas Documentation]: https://netflix.github.io/atlas-docs/spectator/lang/go/usage/

## Local Development

Install a recent version of Go, possibly with [Homebrew](https://brew.sh/).

```shell
make test
make test/cover
```
