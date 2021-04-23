#!/bin/sh
export GIN_MODE=release
[ -d logs ] || mkdir logs
./chat-server 2>&1 | tee ./logs/log-`date +"%Y-%m-%d-%H-%M-%S"`.txt