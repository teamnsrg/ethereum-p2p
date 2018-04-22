#!/bin/bash
# This script assumes required docker images are already built.
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: sudo ./run-no-docker.sh num-instance"
  exit 1
fi

n=$(( $1 - 1 ))
WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
TXSNIPER_NAME="geth-tx-sniper"

# get env variables
source .env

cd ${WORKING_DIR}/..
make geth
cp ${WORKING_DIR}/../build/bin/geth /usr/bin/geth

cd ${WORKING_DIR}
DATADIR="${ROOT_DIR}/${TXSNIPER_NAME}"
mkdir -p -m 755 ${DATADIR}

# run tx-snipers
for i in `seq 0 ${n}`;
do
  ./tx-sniper-loop.sh ${i} >>${DATADIR}/${TXSNIPER_NAME}-${i}-loop.log 2>&1 &
  echo "${TXSNIPER_NAME}-${i} loop started"
done
