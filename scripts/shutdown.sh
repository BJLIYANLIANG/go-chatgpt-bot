#!/bin/bash

export PATH=$PATH:/sbin:/usr/sbin:/usr/local/bin:/usr/bin:/bin:/usr/local/sbin

CURFILE=$(readlink -f "$0")

CURDIR=$(dirname $CURFILE)

MODULE_DIR=${CURDIR%"/scripts"}

MODULE=$(basename $MODULE_DIR)

LogPath=${MODULE_DIR}/log/${MODULE}.op.log

if [[ ! -f "$MODULE_DIR/bin/$MODULE" ]];then
    MODULE=$(echo $MODULE | sed 's/[0-9]*$//')
    if [[ ! -f "$MODULE_DIR/bin/$MODULE" ]];then
        echo "unsupport proc"
        exit 1
    fi
fi

LOCKFILE=${CURDIR}/.lock

function Log() {
  echo -e "$(date +"%Y%m%d %H:%M:%S")\t$(basename $0)\t$*" | tee -a $LogPath
}

function GetLock() {
    if [ -e ${LOCKFILE} ] && kill -0 `cat ${LOCKFILE}` 2> /dev/null; then
        Log "cannot run mutiple $0 at the same time"
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

GetLock

RUNNING_PID=$(GetRunningPID)

if [[ -z "$RUNNING_PID" ]];then
    Log "${MODULE} not running!"
    exit 0
fi

kill ${RUNNING_PID}

MAX_RETRY=600

for ((i=1; i<=$MAX_RETRY; i++))
do
    RUNNING_PID=$(GetRunningPID)

    if [[ -z "$RUNNING_PID" ]];then
        exit 0
    fi

    if [[ "$i" -lt $MAX_RETRY ]];then
        Log "${MODULE} still running pid=$RUNNING_PID, recheck 0.5 seconds later!"
        sleep 0.5
    else
        Log "stop ${MODULE} failed, pid=$RUNNING_PID"
        exit 1
    fi
done
