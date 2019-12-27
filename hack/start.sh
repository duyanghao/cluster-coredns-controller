#!/bin/bash

server="./build/cluster-coredns-controller/cluster-coredns-controller"
let item=0
item=`ps -ef | grep $server | grep -v grep | wc -l`

if [ $item -eq 1 ]; then
	echo "The cluster-coredns-controller is running, shut it down..."
	pid=`ps -ef | grep $server | grep -v grep | awk '{print $2}'`
	kill -9 $pid
fi

echo "Start cluster-coredns-controller now ..."
make src.build
./build/cluster-coredns-controller/cluster-coredns-controller -c ./examples/config.yaml -logtostderr=true -v=5 >> ./cluster-coredns-controller.log 2>&1 &
