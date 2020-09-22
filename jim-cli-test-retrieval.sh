#! /bin/bash

set -x

time lotus client retrieve --miner=t01000 bafk2bzaceadshj2ri6umifoo4bhwvzhqe3fe22hgslw76ovro4pre22zjmcbi /home/ubuntu/downloads/output-$(date +'%s').txt
