#!/bin/sh

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

function GetRunningPID() {
    PIDS=""
    SUSPECT_PIDS=$(ps -fle | grep "\./$MODULE" | grep -v grep | awk '{print $4}')
    for pid in $SUSPECT_PIDS;do
        EXE=$(readlink -f /proc/$pid/exe)
        # read exe path failed, may have no enough permission, just add in legal pid list
        if [[ "$?" -ne 0 ]];then
            PIDS="$PIDS $pid"
        elif [[ "$EXE" == "${MODULE_DIR}/bin/$MODULE" ]] || [[ "$EXE" == "${MODULE_DIR}/bin/$MODULE (deleted)" ]]; then
            PIDS="$PIDS $pid"
        fi
    done
    echo $PIDS
}

PIDS=$(GetRunningPID)

if [[ -n "$PIDS" ]];then
    ps -fl $PIDS
else
    echo "no running process"
    exit 1
fi

