# Tyche API Reference

https://tyche.polispay.com

## HTTP Return Codes
* HTTP `401`: Failed authorization on request.

## Security Type
* PPAT
* MVT

## Endpoints

### Prepare
Requests Shift shift data. Creates a shift order in the serves which is saved for 5 minutes. During this time window the rate contained in the response will be guaranteed.
``
```
POST prepare/
```

Request Body

| Name       | Type   | Required | Description                                                                                       |
|------------|--------|----------|---------------------------------------------------------------------------------------------------|
| from_coin  | string | Y        | The input coin converting from.                                                                   |
| to_coin    | string | Y        | The target coin to convert to.                                                                    |
| amount     | int    | Y        | Amount of from_coin to exchange in satoshis.                                                      |
| to_address | string | Y        | The address that will receive the converted amount. Address must correspond to a to_coin address. |
