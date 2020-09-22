#! /bin/bash

(
	echo Waiting for API...
	lotus wait-api
	./jim-connect.sh
) &

./lotus daemon --genesis=devgen.car --bootstrap=false 2>&1 | tee -a ~/tmp/node.log

