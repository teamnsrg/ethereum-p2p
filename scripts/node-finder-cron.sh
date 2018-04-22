#!/bin/bash
# This script assumes:
# - it's run from the directory it's in.
# - duplicate cronjobs are handled separately.
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: ./node-finder-cron.sh num-instance"
  exit 1
fi

WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
NODEFINDER_NAME="geth-node-finder"
MINUTE="0"
HOUR="0"
n=$(( $1 - 1 ))
CRONJOBS="$(crontab -l 2>/dev/null)"
for i in `seq 0 ${n}`;
do
  CRONJOBS="${CRONJOBS}\n0 0 * * * cd ${WORKING_DIR} && ./node-finder-logrotate.sh ${i}"
done
CRONJOBS="${CRONJOBS}\n0 6 * * * cd ${WORKING_DIR} && ./combine-logs.sh"
echo -e "${CRONJOBS}" | crontab -
echo "${NODEFINDER_NAME} logrotate cronjobs added"
