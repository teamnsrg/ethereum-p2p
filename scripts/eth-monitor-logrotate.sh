#!/bin/bash
# this script should be added to root crontab for each instance as following:
# 0 0 * * * cd /path/to/gitrepo/scripts && ./eth-monitor-logrotate.sh instance-number
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: ./eth-monitor-logrotate.sh instance-number"
  exit 1
fi

i=$1
WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# get env variables
source .env

ETHMONITOR_NAME="geth-eth-monitor"
DATADIR="${ROOT_DIR}/${ETHMONITOR_NAME}/${i}"
LOGDIR="${DATADIR}/${ETHMONITOR_NAME}/logs"
NEWLOGDIR="${ARCHIVE_DIR}/${ETHMONITOR_NAME}/${i}"
[ -d "${NEWLOGDIR}" ] || mkdir -p -m 755 ${NEWLOGDIR}
if cd ${LOGDIR} ; then
  [ -d old ] || mkdir -p -m 755 old
  mv *.log old
  docker exec ${ETHMONITOR_NAME}-${i} geth attach --exec 'admin.logrotate()'
  DATE=$(date -u +%Y%m%dT%H%M%S)
  cd old
  for FILENAME in *.log; do
    mv ${FILENAME} ${FILENAME}-${DATE}Z
  done
  mv * ${NEWLOGDIR}
else
  echo "logdir ${LOGDIR} doesn't exist"
fi
