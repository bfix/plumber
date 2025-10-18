#!/bin/bash

go build -v ./cmd/plumber
./plumber -p rules/plan9 &
export PLUMBER_PID=$!
sleep 1s
9pfuse 127.0.0.1:3124 /mnt/plumb
[ $? -eq 0 ] && rc
kill $PLUMBER_PID
