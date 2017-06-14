#!/bin/bash

IP=127.0.0.1
if [ $# -gt 0 ]; then
  IP=$1
fi
cd "$(dirname "$0")"
./ssh_wrapper -p 2222 root@$IP "cd /; tar zc repos > repos.tgz"
scp -F ./ssh-config -i ./git_pubkey_rsa -P 2222 root@$IP:/repos.tgz .
