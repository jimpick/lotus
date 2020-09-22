#! /bin/bash

lotus net connect $(ssh 192.168.240.128 ~/lotus/lotus net listen | grep 192)
lotus net connect $(ssh 192.168.240.128 ~/lotus/lotus-miner net listen | grep 192)
