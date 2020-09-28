#! /bin/bash

#time lotus client retrieve --miner=t01000 bafk2bzaceadshj2ri6umifoo4bhwvzhqe3fe22hgslw76ovro4pre22zjmcbi /home/ubuntu/downloads/output-$(date +'%s').txt

MINER=t01000
CID=bafk2bzaceadshj2ri6umifoo4bhwvzhqe3fe22hgslw76ovro4pre22zjmcbi
DEST=/home/ubuntu/downloads/output-$(date +'%s').txt
echo "Downloading to: $DEST"

OFFER="$(curl -X POST  -H "Content-Type: application/json"  -H "Authorization: Bearer $(cat ~/.lotus/token)"  --data "{ \"jsonrpc\": \"2.0\", \"method\": \"Filecoin.ClientMinerQueryOffer\", \"params\": [\"$MINER\", { \"/\": \"$CID\" }, null], \"id\": 1 }"  'http://127.0.0.1:1234/rpc/v0')"
#echo $OFFER | jq
ROOT=$(echo $OFFER | jq .result.Root)
SIZE=$(echo $OFFER | jq .result.Size)
MIN_PRICE=$(echo $OFFER | jq .result.MinPrice)
UNSEAL_PRICE=$(echo $OFFER | jq .result.UnsealPrice)
PAYMENT_INTERVAL=$(echo $OFFER | jq .result.PaymentInterval)
PAYMENT_INTERVAL_INCREASE=$(echo $OFFER | jq .result.PaymentIntervalIncrease)
CLIENT=$(lotus wallet default)
MINER_OWNER=$(echo $OFFER | jq .result.Miner)
MINER_PEER=$(echo $OFFER | jq .result.MinerPeer)

ORDER="{ \"Root\": $ROOT, \"Piece\": null, \"Size\": $SIZE, \"Total\": $MIN_PRICE, \"UnsealPrice\": $UNSEAL_PRICE, \"PaymentInterval\": $PAYMENT_INTERVAL, \"PaymentIntervalIncrease\": $PAYMENT_INTERVAL_INCREASE, \"Client\": \"$CLIENT\", \"Miner\": $MINER_OWNER, \"MinerPeer\": $MINER_PEER }"
echo $ORDER | jq

FILEREF="{ \"Path\": \"$DEST\", \"IsCAR\": false }"
DATA="{ \"jsonrpc\": \"2.0\", \"method\": \"Filecoin.ClientRetrieve\", \"params\": [ $ORDER, $FILEREF ], \"id\": 1 }"

curl -X POST -H "Content-Type: application/json" \
       	-H "Authorization: Bearer $(cat ~/.lotus/token)" \
       	--data "$DATA" \
       	'http://127.0.0.1:1238/rpc/v0' 
