#! /bin/bash

curl -X POST  -H "Content-Type: application/json"  -H "Authorization: Bearer $(cat ~/.lotus/token)"  --data '{ "jsonrpc": "2.0", "method": "Filecoin.ClientTestRetrieve", "params": [], "id": 1 }'  'http://127.0.0.1:1234/rpc/v0' 
