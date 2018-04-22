#!/bin/bash
# this script should be added to root crontab for each instance as following:
# */3 * * * * cd /path/to/gitrepo/scripts && ./tx-sniper-peerlist.sh instance-number
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: ./tx-sniper-peerlist.sh instance-number"
  exit 1
fi

i=$1
WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# get env variables
source .env

TXSNIPER_NAME="geth-tx-sniper"
DATADIR="${ROOT_DIR}/${TXSNIPER_NAME}/${i}"
LOGDIR="${DATADIR}/${TXSNIPER_NAME}/logs"
NEWLOGDIR="${ARCHIVE_DIR}/${TXSNIPER_NAME}/${i}/peerlists"
[ -d "${NEWLOGDIR}" ] || mkdir -p -m 755 ${NEWLOGDIR}
if cd ${LOGDIR} ; then
  FILENAME="peerlist.log-$(date -u +%Y%m%dT%H%M%S)Z"
  docker exec ${TXSNIPER_NAME}-${i} geth attach --exec 'admin.peerList' > ${FILENAME}
  mv ${FILENAME} ${NEWLOGDIR}
else
  echo "logdir ${LOGDIR} doesn't exist"
fi
