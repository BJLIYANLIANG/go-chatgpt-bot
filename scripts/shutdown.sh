#!/bin/bash

#关闭服务
cd `dirname $0`/..
export BASE_DIR=`pwd`
pid=`ps ax | grep -i chatgpt-bot | grep "${BASE_DIR}" | grep -v grep | awk '{print $1}'`
if [ -z "$pid" ] ; then
        echo "No chatgpt-bot running."
        exit -1;
fi

echo "The chatgpt-bot(${pid}) is running..."

kill ${pid}

echo "Send shutdown request to chatgpt-bot(${pid}) OK"
