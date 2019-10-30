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

The payload data must be an encrypted JWE, using your firebase user id.

For this endpoint, the data must cointain the following fields:

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
  "token": "J3UU0yu3MNapSMXEE4FvOmuPbpWflbmUo-KR1sFR8jk=",
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

#### New Shift

A shift must be prepared before using this endpoint.

```
POST /new
```

You must add your firebase token to the header, in the "token" field, and the payload must be encrypted using your firebase id.

Example Body:

```javascript
{
    "payload": "eyJhbGciOiJSUzI1NiIsImtpZCI6ImEwYjQwY2NjYmQ0OWQxNmVkMjg2MGRiNzIyNmQ3NDZiNmZhZmRmYzAiLCJ0eXAiOiJKV1QifQ.eyJpc3MiOiJodHRwczovL3NlY3VyZXRva2VuLmdvb2dsZS5jb20vcG9saXNwYXktY29wYXkiLCJhdWQiOiJwb2xpc3BheS1jb3BheSIsImF1dGhfdGltZSI6MTU2OTQzMTY5NSwidXNlcl9pZCI6ImRDdGNxOU00SkdNbzVUcmFXdjJHaGtZY2xIUjIiLCJzdWIiOiJkQ3RjcTlNNEpHTW81VHJhV3YyR2hrWWNsSFIyIiwiaWF0IjoxNTcyMjg0OTY5LCJleHAiOjE1NzIyODg1NjksImVtYWlsIjoiZXJvc0Bwb2xpc3BheS5vcmciLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwiZmlyZWJhc2UiOnsiaWRlbnRpdGllcyI6eyJlbWFpbCI6WyJlcm9zQHBvbGlzcGF5Lm9yZyJdfSwic2lnbl9pbl9wcm92aWRlciI6InBhc3N3b3JkIn19.TWwGaKQXC6XuPIfgceClcvxQV0mQpeFeSCH1D6S07EDwQoJzj_-xBRxO-tH-9m92C6-Jq0gFjSz31hfOGtBwqToTrrEFp0-7a6TPc40yOVPIj_XTuaqixsHrUGhlgi8grYQv8SwfNSkalUSTir5D09CS1RQuU0UcsHJcwY0R5D7U8rh859JioXszNG8MaEhPU6evZVzbW6C_J5erCY-H75K9v0t2XsrSAruL0pKuMrUGRvyDtHa9XTkBDoj9IqKC14YjeTNtV8yMx956XqyPSIk3Ui0U2yi3ZA4pInC2is1ZHqWR02j-3dTQJ8ZDszguZkz3Erbv9y7EWlcb8_jzdg"
}
```

The payload data must be an encrypted JWE, using your firebase user id.

For this endpoint, the data must cointain the following fields:

```javascript
{
  "raw_tx": "020000000178799f3fa63196307c3934d74cd1d16b545a8748e5069999ed08d8ca48023b75010000006b483045022100bdd514ad0e7353dd4dc6a91a1776bfa1b5eaccf4055b19c57bccfca90398aae8022023e16027e0be8600a4868056f951c4b1685cb82709468ac116c9cf37c97e693d012103a6bb9cb760d2ed2fe95b7d79e00232011cf946e8fc06ab22d17739040e6f5c25ffffffff020065cd1d000000001976a9147931bed12188f4e7a7a75d1e7a1618ffa714c56488ac7b90e05d000000001976a9140d58f8b00393b0b2c475cfc9f56d7ebe0fd7c6ee88ac00000000",
  "fee_tx": "020000000178799f3fa63196307c3934d74cd1d16b545a8748e5069999ed08d8ca48023b75010000006b483045022100f2bb4b4e25d4f719cd94c25a4c899e1829abda7b79c32f749a42765f750cdc8902206946f7a0312e11b8ebea78692425da3e1c5e829fb4adb3de2d2ee8ade17a4f96012103a6bb9cb760d2ed2fe95b7d79e00232011cf946e8fc06ab22d17739040e6f5c25ffffffff022f304c00000000001976a914236ed091469a7ccc740f24633bfad43a695828ab88ac4cc5617b000000001976a9140d58f8b00393b0b2c475cfc9f56d7ebe0fd7c6ee88ac00000000",
  "token": "J3UU0yu3MNapSMXEE4FvOmuPbpWflbmUo-KR1sFR8jk=",
  "pay_address": "32EVPiYobns5sKscbThgaVJBAHd5q1dxii"
}
```

Example Decrypted Response:

```javascript
{
  "status": "success",
}
```

## Testing

Simply run:

```
go test ./...
```

## Contributing

Pull requests accepted.
