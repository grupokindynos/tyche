# Tyche

> Tyche is the Greek goddess of chance, fate and fortune

![Actions](https://github.com/grupokindynos/tyche/workflows/Tyche/badge.svg)
[![codecov](https://codecov.io/gh/grupokindynos/tyche/branch/master/graph/badge.svg)](https://codecov.io/gh/grupokindynos/tyche)[![Go Report](https://goreportcard.com/badge/github.com/grupokindynos/tyche)](https://goreportcard.com/report/github.com/grupokindynos/tyche)
[![GoDocs](https://godoc.org/github.com/grupokindynos/tyche?status.svg)](http://godoc.org/github.com/grupokindynos/tyche)

Tyche is a microservice API for shifting between cryptocurrencies

## Building

To run Tyche simply clone the repository:

```
git clone https://github.com/grupokindynos/tyche
```

Build it or Run it:

```
go build && ./tyche
```

```
go run main.go
```

## Running flags
```
-local
```

Set this flag to run tyche using the testing hestia database. Default is false (production mode).
When using this flag you must be running hestia locally on port 8080.

```
-port=xxxx
```

Specifies the running port. Default is 8080 

```
-stop-proc
```

Set this flag to run tyche without processor.

```
-no-txs
```

Set this flag to avoid publishing txs on the blockchain but store them on the database.
WARNING: -local flag must be set in order to use this flag.

```
-skip-val
```

Set this flag to skip validations on txs (currently just skipping the minimum amount of confirmations required to process a tx)
WARNING: -local flag must be set in order to use this flag.

## API Reference

@TODO

## Testing

Simply run:

```
go test ./...
```

## Contributing

Pull requests accepted.
