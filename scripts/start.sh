#!/bin/bash
#后台运行chatgpt-bot执行脚本

export BASE_DIR=`pwd`
echo $BASE_DIR

# check the nohup.out log output file
if [ ! -f "${BASE_DIR}/nohup.out" ]; then
  touch "${BASE_DIR}/nohup.out"
echo "create file  ${BASE_DIR}/nohup.out"
fi

nohup "${BASE_DIR}/chatgpt-bot" start -c chatgpt.json & tail -f "${BASE_DIR}/nohup.out"

echo "chatgpt-bot is starting，you can check the ${BASE_DIR}/nohup.out"
