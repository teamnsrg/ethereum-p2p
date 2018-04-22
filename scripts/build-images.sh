#!/bin/bash
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi

WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# build tx-sniper container image
cd ${WORKING_DIR}/..
docker build -t "geth:tx-sniper" .
