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
  echo "usage: ./tx-sniper-cron.sh num-instance"
  exit 1
fi

WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
TXSNIPER_NAME="geth-tx-sniper"
MINUTE="0"
HOUR="*"
n=$(( $1 - 1 ))
CRONJOBS="$(crontab -l 2>/dev/null)"
for i in `seq 0 ${n}`;
do
  CRONJOBS="${CRONJOBS}\n0 * * * * cd ${WORKING_DIR} && ./tx-sniper-logrotate.sh ${i}"
  CRONJOBS="${CRONJOBS}\n*/3 * * * * cd ${WORKING_DIR} && ./tx-sniper-peerlist.sh ${i}"
done
echo -e "${CRONJOBS}" | crontab -
echo "${TXSNIPER_NAME} cronjobs added"
