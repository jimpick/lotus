#! /bin/bash

killall lotus
rm -rf ~/.lotus ~/.lotusminer ~/.genesis-sectors ~/tmp/*.log localnet.json *.car
./lotus-seed pre-seal --sector-size 2KiB --num-sectors 2
./lotus-seed genesis new localnet.json
./lotus-seed genesis add-miner localnet.json ~/.genesis-sectors/pre-seal-t01000.json
#./lotus daemon --lotus-make-genesis=devgen.car --genesis-template=localnet.json --bootstrap=false

echo "./lotus daemon --lotus-make-genesis=devgen.car --genesis-template=localnet.json --bootstrap=false 2>&1 | tee -a ~/tmp/node.log"
