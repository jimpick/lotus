#! /bin/bash

set -e

PARAMS=($@)

NUM=${PARAMS[0]}

#if [ "$NUM" != "0" -a "$NUM" != "1" -a "$NUM" != "2" ]; then
#  echo Wrong value for NUM: $NUM
#  exit 1
#fi

URL_NODE=https://lotus.testground.ipfs.team/api/$NUM/testplan/.lotus/token
TOKEN_NODE=$(curl -s $URL_NODE)

URL_MINER=https://lotus.testground.ipfs.team/api/$NUM/testplan/.lotusminer/token
TOKEN_MINER=$(curl -s $URL_MINER)

export FULLNODE_API_INFO=$TOKEN_NODE:/ip4/127.0.0.1/tcp/$((11234 + $NUM))/http
export MINER_API_INFO=$TOKEN_MINER:/ip4/127.0.0.1/tcp/$((12345 + $NUM))/http

echo Node: $NUM - lotus-miner ${PARAMS[*]:1}
echo Node API Port: $((11234 + $NUM))
echo Miner API Port: $((12345 + $NUM))

./lotus-miner ${PARAMS[*]:1}
