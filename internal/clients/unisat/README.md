# Unisat Client

In Babylon, we use the Ordinal Service (https://github.com/ordinals/ord) to check 
if UTXOs contain any BRC-20/Runes/Ordinals. In addition to the Ordinal Service, 
we also use the paid service from Unisat to double-check the UTXOs in case of 
Ordinal Service downtime.

You can find more information about Unisat's Ordinal/BRC-20/Runes related endpoints at:
https://docs.unisat.io/

In our service, we only utilize the following endpoint:
- `/v1/indexer/address/{{address}}/inscription-utxo-data`

## How to Use It

1. Log in via https://developer.unisat.io/account/login (create an account if you don't have one).
2. Copy the `API-Key`.
3. Set the key as an environment variable named `UNISAT_TOKEN`.
4. Configure the values for `unisat.host`, `limit`, `timeout`, etc. Refer to `config-docker.yml`.
5. Ensure you also set up the `ordinals` configuration, as this is a dependency.
6. Call the POST endpoint `/v1/ordinals/verify-utxos` as shown in the example below:
7. The calls to unisat will only be triggered if ordinal service is not responding or returning errors

```POST
{
    "utxos": [
        {
            "txid": "143c33b4ff4450a60648aec6b4d086639322cb093195226c641ae4f0ae33c3f5",
            "vout": 2
        },
        {
            "txid": "be3877c8dedd716f026cc77ef3f04f940b40b064d1928247cff5bb08ef1ba58e",
            "vout": 0
        },
        {
            "txid": "d7f65a37f59088b3b4e4bc119727daa0a0dd8435a645c49e6a665affc109539d",
            "vout": 0
        }
    ],
    "address": "tb1pyqjxwcdv6pfcaj2l565ludclz2pwu2k5azs6uznz8kml74kkma6qm0gzlv"
}
