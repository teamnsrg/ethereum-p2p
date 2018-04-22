#!/bin/bash
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: sudo ./tx-sniper-loop.sh instance-number"
  exit 1
fi

i=$1
WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd ${WORKING_DIR}
source .env
TXSNIPER_NAME="geth-tx-sniper"
URL="research-scan.sprai.org"
TXSNIPER_PORT=30403
PORT=$(( ${TXSNIPER_PORT}+${i} ))
DATADIR="${ROOT_DIR}/${TXSNIPER_NAME}/${i}"
mkdir -p -m 755 ${DATADIR}/${TXSNIPER_NAME}
ERRFILE="${DATADIR}/${TXSNIPER_NAME}-error.log"
IDENTITY="uiuc-${i}(${URL})"
SLEEP=3
echo "starting ${TXSNIPER_NAME}-${i}..."
while true
do
  geth \
    --identity "${IDENTITY}" \
    --datadir "${DATADIR}" \
    --port ${PORT} \
    --verbosity 5 \
    --logtofile \
    --maxnumfile 1048576 \
    --nodiscover \
    --nolisten >>${ERRFILE} 2>&1
  echo "${TXSNIPER_NAME}-${i} stopped. restarting in ${SLEEP} seconds..."
  sleep ${SLEEP}
  echo "restarting ${TXSNIPER_NAME}-${i}..."
done
