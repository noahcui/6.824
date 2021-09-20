#!/usr/bin/env bash

PID_FILE=server.pid

PID=$(cat "${PID_FILE}");

if [ -z "${PID}" ]; then
    echo "Process id for servers is written to location: {$PID_FILE}"
    #go build ../cmd/
    rm -r logs
    mkdir logs/
    ./test.sh &
    echo $! >> ${PID_FILE}
else
    echo "Servers are already started in this folder."
    exit 0
fi