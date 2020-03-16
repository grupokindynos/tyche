# Tyche Open API Reference

https://tyche.polispay.com/shift/open/

## HTTP Return Codes WIP
* HTTP `401`: Failed authorization on request.

## Authentication
Use basic auth to use the API. The development team will provide you with the necessary credentials.

## Encryption
@TODO

## Endpoints WIP

### Prepare
Requests Shift shift data. Creates a shift order in the serves which is saved for 5 minutes. During this time window the rate contained in the response will be guaranteed.

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


Response
```
{
    "fee": {
        "address": "polis address",
        "amount": 19082983,
        "has_fee": false
    },
    "payment": {
        "address": "polis address",
        "amount": 19082983,
        "has_fee": false
    },
    "received_amount": 37465434,
    "shift_id": "unique_id"
}
```

### Store
Confirms prepared shift data. Requests server to store and start processing the shift. Returns the id of the stored shift for troubleshooting requests.

```
POST new/
```

Request Body

| Name       | Type   | Required | Description                                                                                       |
|------------|--------|----------|---------------------------------------------------------------------------------------------------|
| raw_tx  | string | Y        | The serialized transaction for the payment.                                                                   |
| fee_tx    | string | Y        | The serialized transaction for the fee payment.                                                                    |
| refund_addr     | string    | Y        | Address corresponding to the from_coin, to use in case a refund is due                                    |
| has_fee | boolean | Y        | The fee boolean received in the prepare request. |
| shift_id | string | Y        | The shift id that was returned when calling the prepare/ endpoint. Shift will be available for 5 minutes after the prepare/ response is returned. |


Response
```
{
    "shiftid":"uuid"
}
```

### Balance
Retrieves available amounts on the Hot Wallets.

```
GET balance/:coin
```

Parameters

| Name       | Required | Description                                                                                       |
|------------|----------|---------------------------------------------------------------------------------------------------|
| coin  | Y        | The serialized transaction for the payment.                                                                   |


Response
```
{
    "confirmed": 0.00000000,
    "unconfirmed": 0.00000000
}
```

### Status
Retrieves the status of the service.

```
GET status/
```

Response
```
{
    "service": true
}
```

## Error Handling
