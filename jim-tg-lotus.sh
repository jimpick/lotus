#! /bin/bash

set -e

PARAMS=($@)

NUM=${PARAMS[0]}

#if [ "$NUM" != "0" -a "$NUM" != "1" -a "$NUM" != "2" ]; then
#  echo Wrong value for NUM: $NUM
#  exit 1
#fi

URL=https://lotus.testground.ipfs.team/api/$NUM/testplan/.lotus/token

TOKEN=$(curl -s $URL)

export FULLNODE_API_INFO=$TOKEN:/ip4/127.0.0.1/tcp/$((11234 + $NUM))/http

echo Node: $NUM - lotus ${PARAMS[*]:1}

./lotus ${PARAMS[*]:1}
