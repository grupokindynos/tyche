# Tyche

> Tyche is the Greek goddess of chance, fate and fortune

[![CircleCI](https://circleci.com/gh/grupokindynos/plutus.svg?style=svg)](https://circleci.com/gh/grupokindynos/plutus)
[![codecov](https://codecov.io/gh/grupokindynos/plutus/branch/master/graph/badge.svg)](https://codecov.io/gh/grupokindynos/plutus)[![Go Report](https://goreportcard.com/badge/github.com/grupokindynos/plutus)](https://goreportcard.com/report/github.com/grupokindynos/plutus)
[![GoDocs](https://godoc.org/github.com/grupokindynos/plutus?status.svg)](http://godoc.org/github.com/grupokindynos/plutus)

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

Make sure the port is configured under en enviroment variable `PORT=8080`

## API Reference

#### Prepare Shift

A shift must be prepared before being submitted.

```
POST /prepare
```

You must add your firebase token to the header, in the "token" field, and the payload must be encrypted using your firebase id.

Example Body:

```javascript
{
    "payload": "eyJhbGciOiJSUzI1NiIsImtpZCI6ImEwYjQwY2NjYmQ0OWQxNmVkMjg2MGRiNzIyNmQ3NDZiNmZhZmRmYzAiLCJ0eXAiOiJKV1QifQ.eyJpc3MiOiJodHRwczovL3NlY3VyZXRva2VuLmdvb2dsZS5jb20vcG9saXNwYXktY29wYXkiLCJhdWQiOiJwb2xpc3BheS1jb3BheSIsImF1dGhfdGltZSI6MTU2OTQzMTY5NSwidXNlcl9pZCI6ImRDdGNxOU00SkdNbzVUcmFXdjJHaGtZY2xIUjIiLCJzdWIiOiJkQ3RjcTlNNEpHTW81VHJhV3YyR2hrWWNsSFIyIiwiaWF0IjoxNTcyMjg0OTY5LCJleHAiOjE1NzIyODg1NjksImVtYWlsIjoiZXJvc0Bwb2xpc3BheS5vcmciLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwiZmlyZWJhc2UiOnsiaWRlbnRpdGllcyI6eyJlbWFpbCI6WyJlcm9zQHBvbGlzcGF5Lm9yZyJdfSwic2lnbl9pbl9wcm92aWRlciI6InBhc3N3b3JkIn19.TWwGaKQXC6XuPIfgceClcvxQV0mQpeFeSCH1D6S07EDwQoJzj_-xBRxO-tH-9m92C6-Jq0gFjSz31hfOGtBwqToTrrEFp0-7a6TPc40yOVPIj_XTuaqixsHrUGhlgi8grYQv8SwfNSkalUSTir5D09CS1RQuU0UcsHJcwY0R5D7U8rh859JioXszNG8MaEhPU6evZVzbW6C_J5erCY-H75K9v0t2XsrSAruL0pKuMrUGRvyDtHa9XTkBDoj9IqKC14YjeTNtV8yMx956XqyPSIk3Ui0U2yi3ZA4pInC2is1ZHqWR02j-3dTQJ8ZDszguZkz3Erbv9y7EWlcb8_jzdg"
}
```

In the decrypted data, you must send the coin you are sending from, the coin you are converting too, the coin you will be paying the fee with, and the amount in satoshis.
Example decrypted data:

```javascript
{
  "from_coin": "POLIS",
  "to_coin": "BTC",
  "fee_coin": "POLIS",
  "amount": 50000000
}
```

Example Decrypted Response:

```javascript
{
  "token": "ed8f3af8c10ca70a136901c6dd3adf037f0aea8a93fbe9e8014034300f1e",
  "timestamp": 1572407136,
  "rate": {
      "rate": 123123,
      "from_coin": "POLIS",
      "from_amount": 50000000,
      "to_coin": "BTC",
      "to_amount": 30000000,
      "to_address": "PE7km6uoXJYHXxyqzuEvrVYZd1sVz7Egm4"
      "fee_coin": "POLIS"
      "fee_amount": 3000000,
      "fee_address": "PJsDJF5k6dmxxpENCbMGLZ9k6AjXofHViP"
  }
}
```

## Testing

Simply run:

```
go test ./...
```

## Contributing

Pull requests accepted.
