#!/bin/bash

go build -v ./cmd/plumber

RULES=${1:-rules/plan9}
./plumber -p $RULES &
export PLUMBER_PID=$!
sleep 1s
9pfuse 127.0.0.1:3124 /mnt/plumb
[ $? -eq 0 ] && rc
fusermount -u /mnt/plumb
kill $PLUMBER_PID
