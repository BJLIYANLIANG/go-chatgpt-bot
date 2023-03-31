#!/bin/bash

export PATH=$PATH:/sbin:/usr/sbin:/usr/local/bin:/usr/bin:/bin:/usr/local/sbin

CURFILE=$(readlink -f "$0")

CURDIR=$(dirname $CURFILE)

MODULE_DIR=${CURDIR%"/scripts"}

MODULE=$(basename $MODULE_DIR)

if [[ ! -f "$MODULE_DIR/bin/$MODULE" ]];then
    MODULE=$(echo $MODULE | sed 's/[0-9]*$//')
    if [[ ! -f "$MODULE_DIR/bin/$MODULE" ]];then
        echo "unsupport proc"
        exit 1
    fi
fi

CONSOLE_LOG=${MODULE_DIR}/log/${MODULE}.console.log

LOCKFILE=${CURDIR}/.lock

export LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:${MODULE_DIR}/bin

[[ ! -d ${MODULE_DIR}/log ]] && mkdir ${MODULE_DIR}/log

function RotateConsoleLog() {
    if [[ -f "$CONSOLE_LOG" ]];then
        LINE_COUNT=$(wc -l $CONSOLE_LOG | awk '{print $1}')
        MAX_LINE=10000
        if [[ "$LINE_COUNT" -gt "$MAX_LINE" ]];then
            echo '' > $CONSOLE_LOG
        fi
    fi
}

function GetLock() {
    if [ -e ${LOCKFILE} ] && kill -0 `cat ${LOCKFILE}` 2> /dev/null; then
        echo "cannot run mutiple $0 at the same time"
        exit 1
    fi

    trap "rm -f ${LOCKFILE};exit" INT TERM EXIT
    echo $$ > ${LOCKFILE}
}

function GetRunningPID() {
    PIDS=""
    SUSPECT_PIDS=$(ps -fle | grep "\./$MODULE" | grep -v grep | awk '{print $4}')
    for pid in $SUSPECT_PIDS;do
        EXE=$(readlink -f /proc/$pid/exe)
        if [[ "$EXE" == "${MODULE_DIR}/bin/$MODULE" ]] || [[ "$EXE" == "${MODULE_DIR}/bin/$MODULE (deleted)" ]]; then
            if [[ -z "$PIDS" ]];then
                PIDS="$pid"
            else
                PIDS="$PIDS $pid"
            fi
        fi
    done
    echo $PIDS
}

# disable coredump when disk usage > threshold and pre core num > threshold
function SetupCoredumpSetting() {
    MAX_CORE_NUM=3
    CUR_CORE_NUM=$(find /data/coredump -name "core_${MODULE}" | wc -l)

    DISK_THRESHOLD=80
    DISK_USAGE=$(df -lh | awk '{if ($NF=="/data") print $(NF -1)}' | tr -d '%')

    if [[ "$DISK_USAGE" -ge "$DISK_THRESHOLD" ]] && [[ "$CUR_CORE_NUM" -ge "$MAX_CORE_NUM" ]];then
        ulimit -c 0
    else
        ulimit -c unlimited
    fi
}

GetLock

if [[ -f "${CURDIR}/.disable-autorestart" ]];then
    echo "clearn ${CURDIR}/.disable-autorestart to reenable auto restart"
    rm ${CURDIR}/.disable-autorestart
fi

RUNNING_PID=$(GetRunningPID)

if [[ -n "$RUNNING_PID" ]];then
    echo "${MODULE} have already been running pid:$RUNNING_PID, run ./p.sh to show detail!"
    exit 0
fi

SetupCoredumpSetting

RotateConsoleLog

mkdir -p ${MODULE_DIR}/log

cd ${MODULE_DIR}/bin

# clear log
echo '' > ../log/chatgpt-bot.log

ulimit -n 500000
./${MODULE} start --configFile=../etc/chatgpt.json --logFile=../log/chatgpt-bot.log >> ${CONSOLE_LOG} 2>&1 &

STARTED_PID=$(GetRunningPID)

if [[ -z "$STARTED_PID" ]];then
    echo "${MODULE} start failed, have no running ${MODULE} pid"
    exit 1
else
    ps -fl $STARTED_PID
fi

echo "Instance [$STARTED_PID] Start OK!"
sleep 1
sync # 同步日志
cat ../log/chatgpt-bot.log
exit 0

