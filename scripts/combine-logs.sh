#!/bin/bash
# this script should be added to root crontab for each instance as following:
# 0 0 * * * cd /path/to/gitrepo/scripts && ./node-finder-logrotate.sh instance-number
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi

WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# get env variables
source .env

NODEFINDER_NAME="geth-node-finder"
TRIMMED="${ROOT_DIR}/${NODEFINDER_NAME}/trimmed-logs"
PROCESSED="${ROOT_DIR}/${NODEFINDER_NAME}/processed-logs"
[ -d "${PROCESSED}" ] || mkdir -p -m 755 ${PROCESSED}
if cd ${TRIMMED} ; then
  # get unique nodeids
  grep 'DEVP2P' peer-*.txt | cut -d'|' -f2 | sort -uV > ${PROCESSED}/devp2p-id.txt
  grep 'ETHEREUM' peer-*.txt | cut -d'|' -f2 | sort -uV > ${PROCESSED}/ethereum-id.txt
  grep 'MAINNET' peer-*.txt | cut -d'|' -f2 | sort -uV > ${PROCESSED}/mainnet-id.txt

  # 1 instance
  cp hello-0.txt ${PROCESSED}/1-instance-hello.txt
  cp status-0.txt ${PROCESSED}/1-instance-status.txt
  cp task-0.txt ${PROCESSED}/1-instance-task.txt
  cp disc-proto.txt ${PROCESSED}/1-instance-disc-proto.txt
  sort -V {hello,disc-proto}-0.txt > ${PROCESSED}/1-instance-hello-disc-proto.txt
  ${WORKING_DIR}/process-logs.sh 1 &
  # 10 instances
  cat hello-{0..9}.txt > 10-instance-hello.tmp
  sort -V 10-instance-hello.tmp > ${PROCESSED}/10-instance-hello.txt
  cat status-{0..9}.txt > 10-instance-status.tmp
  sort -V 10-instance-status.tmp > ${PROCESSED}/10-instance-status.txt
  cat task-{0..9}.txt > 10-instance-task.tmp
  sort -V 10-instance-task.tmp > ${PROCESSED}/10-instance-task.txt
  cat disc-proto-{0..9}.txt > 10-instance-disc-proto.tmp
  sort -V 10-instance-disc-proto.tmp > ${PROCESSED}/10-instance-disc-proto.txt
  sort -V 10-instance-{hello,disc-proto}.tmp > ${PROCESSED}/10-instance-hello-disc-proto.txt
  mv 10-instance-hello.tmp 20-instance-hello.tmp
  mv 10-instance-status.tmp 20-instance-status.tmp
  mv 10-instance-task.tmp 20-instance-task.tmp
  mv 10-instance-disc-proto.tmp 20-instance-disc-proto.tmp
  ${WORKING_DIR}/process-logs.sh 10 &
  # 20 instances
  cat hello-{10..19}.txt >> 20-instance-hello.tmp
  sort -V 20-instance-hello.tmp > ${PROCESSED}/20-instance-hello.txt
  cat status-{10..19}.txt >> 20-instance-status.tmp
  sort -V 20-instance-status.tmp > ${PROCESSED}/20-instance-status.txt
  cat task-{10..19}.txt >> 20-instance-task.tmp
  sort -V 20-instance-task.tmp > ${PROCESSED}/20-instance-task.txt
  cat disc-proto-{10..19}.txt >> 20-instance-disc-proto.tmp
  sort -V 20-instance-disc-proto.tmp > ${PROCESSED}/20-instance-disc-proto.txt
  sort -V 20-instance-{hello,disc-proto}.tmp > ${PROCESSED}/20-instance-hello-disc-proto.txt
  mv 20-instance-hello.tmp 30-instance-hello.tmp
  mv 20-instance-status.tmp 30-instance-status.tmp
  mv 20-instance-task.tmp 30-instance-task.tmp
  mv 20-instance-disc-proto.tmp 30-instance-disc-proto.tmp
  ${WORKING_DIR}/process-logs.sh 20 &
  # 30 instances
  cat hello-{20..29}.txt >> 30-instance-hello.tmp
  sort -V 30-instance-hello.tmp > ${PROCESSED}/30-instance-hello.txt
  cat status-{20..29}.txt >> 30-instance-status.tmp
  sort -V 30-instance-status.tmp > ${PROCESSED}/30-instance-status.txt
  cat task-{20..29}.txt >> 30-instance-task.tmp
  sort -V 30-instance-task.tmp > ${PROCESSED}/30-instance-task.txt
  cat disc-proto-{20..29}.txt >> 30-instance-disc-proto.tmp
  sort -V 30-instance-disc-proto.tmp > ${PROCESSED}/30-instance-disc-proto.txt
  sort -V 30-instance-{hello,disc-proto}.tmp > ${PROCESSED}/30-instance-hello-disc-proto.txt
  rm 30-instance-{hello,status,task,disc-proto}.tmp
  ${WORKING_DIR}/process-logs.sh 30 &
else
  echo "dir ${TRIMMED} doesn't exist"
fi
