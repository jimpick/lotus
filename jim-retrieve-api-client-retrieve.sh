#! /bin/bash

curl -X POST -H "Content-Type: application/json" \
       	-H "Authorization: Bearer $(cat ~/.lotus/token)" \
       	--data '{ "jsonrpc": "2.0", "method": "Filecoin.Hello", "params": [], "id": 1 }' \
       	'http://127.0.0.1:1238/rpc/v0' 
