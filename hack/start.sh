#!/bin/bash

server="./build/coredns-sync/coredns-sync"
let item=0
item=`ps -ef | grep $server | grep -v grep | wc -l`

if [ $item -eq 1 ]; then
	echo "The coredns-sync is running, shut it down..."
	pid=`ps -ef | grep $server | grep -v grep | awk '{print $2}'`
	kill -9 $pid
fi

echo "Start coredns-sync now ..."
make src.build
./build/coredns-sync/coredns-sync -v=5 -logtostderr=true -c ./examples/config.yml >> ./coredns-sync.log 2>&1 &
