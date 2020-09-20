#! /bin/bash

./lotus daemon --genesis=devgen.car --bootstrap=false 2>&1 | tee -a ~/tmp/node-$(date +'%s').log
