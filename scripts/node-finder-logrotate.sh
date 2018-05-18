#!/bin/bash
# this script should be added to root crontab for each instance as following:
# 0 0 * * * cd /path/to/gitrepo/scripts && ./node-finder-logrotate.sh instance-number
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: ./node-finder-logrotate.sh instance-number"
  exit 1
fi

i=$1
WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# get env variables
source .env

NODEFINDER_NAME="geth-node-finder"
DATADIR="${ROOT_DIR}/${NODEFINDER_NAME}/${i}"
LOGDIR="${DATADIR}/${NODEFINDER_NAME}/logs"
NEWLOGDIR="${ARCHIVE_DIR}/${NODEFINDER_NAME}/${i}"
TRIMMED="${ARCHIVE_DIR}/${NODEFINDER_NAME}/trimmed-logs"
[ -d "${TRIMMED}" ] || mkdir -p -m 755 ${TRIMMED}
[ -d "${NEWLOGDIR}" ] || mkdir -p -m 755 ${NEWLOGDIR}
if cd ${LOGDIR} ; then
  [ -d old ] || mkdir -p -m 755 old
  mv *.log old
  docker exec ${NODEFINDER_NAME}-${i} geth attach --exec 'admin.logrotate()'
  DATE=$(date -u +%Y%m%dT%H%M%S)
  cd old
  cut -d'|' -f3- disc-proto.log | sed 's/:/|/;s/[a-zA-Z]*=//g' >> ${TRIMMED}/disc-proto-${i}.txt
  cut -d'|' -f3- hello.log | sed -E 's/:/|/;s/[0-9a-zA-Z]*(=|:)//g;s/ /|/g' >> ${TRIMMED}/hello-${i}.txt
  cut -d'|' -f3- status.log | sed -E 's/:/|/;s/[0-9a-zA-Z]*(=|:)//g;s/ /|/g' >> ${TRIMMED}/status-${i}.txt
  cut -d'|' -f3- daofork.log | sed -E 's/:/|/;s/[a-zA-Z]*=//g' >> ${TRIMMED}/daofork-${i}.txt
  grep 'NEW' task.log | awk -F'|' '{print $2"|"$5"|"$6"|"$4"|"$7}' | sed 's/:/|/;s/[a-zA-Z]*=//g' | grep -v 'wait' >> ${TRIMMED}/task-${i}.txt
  grep 'ADD' peer.log | cut -d'|' -f2,4- | sed 's/:/|/;s/[a-zA-Z]*=//g' >> ${TRIMMED}/peer-${i}.txt
  for FILENAME in *.log; do
    mv ${FILENAME} ${FILENAME}-${DATE}Z
  done
  mv * ${NEWLOGDIR}
  cd ${WORKING_DIR} && ./combine-logs.sh &
else
  echo "logdir ${LOGDIR} doesn't exist"
fi
