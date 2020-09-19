#! /bin/bash

curl -X POST  -H "Content-Type: application/json"  -H "Authorization: Bearer $(cat ~/.lotus/token)"  --data '{ "jsonrpc": "2.0", "method": "Filecoin.ClientFindData", "params": [{ "/": "bafk2bzaceadshj2ri6umifoo4bhwvzhqe3fe22hgslw76ovro4pre22zjmcbi" }, null], "id": 1 }'  'http://127.0.0.1:1234/rpc/v0' | jq -C . | less -RM
