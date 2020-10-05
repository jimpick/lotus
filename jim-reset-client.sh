#! /bin/bash

killall lotus 2> /dev/null
rm -rf ~/.lotus ~/tmp/*.log

./jim-copy-car.sh

#echo "./lotus daemon --genesis=devgen.car --bootstrap=false 2>&1 | tee -a ~/tmp/node.log"
echo ./jim-run-client.sh
