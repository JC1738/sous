#!/bin/bash

IP=127.0.0.1
if [ $# -gt 0 ]; then
  IP=$1
fi
cd "$(dirname "$0")"
./ssh_wrapper -p 2222 root@$IP "/reset-repos"
