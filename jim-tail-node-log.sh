#! /bin/bash

tail -1000f `ls -t ~/tmp/node-*.log | head -1` | grep -i jim
