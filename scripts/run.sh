#!/bin/bash
# This script assumes required docker images are already built.
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: sudo ./run.sh num-instance"
  exit 1
fi

n=$(( $1 - 1 ))
WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
TXSNIPER_NAME="geth-tx-sniper"

# get env variables
source .env

TXSNIPER_IMAGE="geth:tx-sniper"

# run tx-snipers
URL="research-scan.sprai.org"
TXSNIPER_PORT=30403
DATADIR="/root/.ethereum"
echo "starting ${TXSNIPER_NAME} containers..."
for i in `seq 0 ${n}`;
do
  IDENTITY="uiuc-${i}(${URL})"
  TXSNIPER_DIR="${ROOT_DIR}/${TXSNIPER_NAME}/${i}"
  [ -d "${TXSNIPER_DIR}" ] || mkdir -p -m 755 ${TXSNIPER_DIR}/${TXSNIPER_NAME}
  PORT=$(( ${TXSNIPER_PORT}+${i} ))
  CMD="geth \
    --identity \"${IDENTITY}\" \
    --datadir \"${DATADIR}\" \
    --port ${PORT} \
    --verbosity 5 \
    --logtofile \
    --maxnumfile 1048576 \
    --nodiscover \
    --nolisten"
  docker run -dit --restart=always -h ${TXSNIPER_NAME}-${i} --name ${TXSNIPER_NAME}-${i} --net host -v ${TXSNIPER_DIR}:${DATADIR} -e CMD="${CMD}" --entrypoint '/bin/sh' ${TXSNIPER_IMAGE} -c "${CMD}"
 echo "${TXSNIPER_NAME}-${i} started"
done
