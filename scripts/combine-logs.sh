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

DATE=$(date '+%Y%m%d')
NODEFINDER_NAME="geth-node-finder"
TRIMMED="${ARCHIVE_DIR}/${NODEFINDER_NAME}/trimmed-logs"
PROCESSED="${ARCHIVE_DIR}/${NODEFINDER_NAME}/processed-logs/${DATE}"
TMP="${PROCESSED}/tmp"
[ -d "${TMP}" ] || mkdir -p -m 755 ${TMP}
if cd ${TRIMMED} ; then
  rm ${PROCESSED}/combined-{hello,disc-proto,status,task,daofork}.txt
  rm ${TMP}/{1..30}-instance-hello-disc-proto.txt
  for i in `seq 0 29`;
  do
    awk -F'|' -v instance="${i}" '{s=index($0,$3);l=index($0,$15)-s-1;print instance"|"$1"|"substr($2,1,16)"|"substr($0,s,l)"|"substr($0,index($0,$15),16)"|"substr($0,index($0,$16),16)}' status-${i}.txt >> ${PROCESSED}/combined-status.txt
    awk -F'|' -v instance="${i}" '{print instance"|"$1"|"substr($2,1,16)"|"$3"|"substr($0, index($0,$4))}' task-${i}.txt >> ${PROCESSED}/combined-task.txt
    awk -F'|' -v instance="${i}" '{print instance"|"$1"|"substr($2,1,16)"|"substr($0, index($0,$3))}' daofork-${i}.txt >> ${PROCESSED}/combined-daofork.txt
    awk -F'|' -v instance="${i}" '{print instance"|"$1"|"substr($2,1,16)"|"substr($0, index($0,$3))}' hello-${i}.txt > ${TMP}/hello.tmp
    cat ${TMP}/hello.tmp >> ${PROCESSED}/combined-hello.txt
    awk -F'|' -v instance="${i}" '{print instance"|"$1"|"substr($2,1,16)"|"substr($0, index($0,$3))}' disc-proto-${i}.txt > ${TMP}/disc-proto.tmp
    cat ${TMP}/disc-proto.tmp >> ${PROCESSED}/combined-disc-proto.txt
    ii=$(( 30 - i ))
    for j in `seq ${ii} 30`;
    do
      cut -d'|' -f1-8 ${TMP}/{hello,disc-proto}.tmp >> ${TMP}/${j}-instance-hello-disc-proto.txt
    done
  done
  rm ${TMP}/{hello,disc-proto}.tmp
  grep 'discover' ${PROCESSED}/combined-task.txt > ${PROCESSED}/combined-task-discover.txt &
  cut -d'|' -f2,3,9 ${PROCESSED}/combined-daofork.txt | sort -t'|' -Vk2,2 -k1nr,1 | sort -t'|' -u -Vk2,2 | cut -d'|' -f2- > ${PROCESSED}/daofork-id-abbr.txt &

  # get unique nodeids
  cut -d'|' -f3 ${TMP}/30-instance-hello-disc-proto.txt | sort -u > ${PROCESSED}/devp2p-id-abbr.txt &
  cut -d'|' -f3 ${PROCESSED}/combined-status.txt | sort -u > ${PROCESSED}/ethereum-id-abbr.txt &
  grep '|1|[0-9]*|[0-9a-zA-Z]*|d4e56740f876aef8$' ${PROCESSED}/combined-status.txt > ${PROCESSED}/combined-status-mainnet.txt
  cut -d'|' -f3 ${PROCESSED}/combined-status-mainnet.txt | sort -u > ${PROCESSED}/mainnet-id-abbr.txt
  grep -Ff ${PROCESSED}/mainnet-id-abbr.txt ${PROCESSED}/combined-hello.txt > ${PROCESSED}/combined-hello-mainnet.txt &
else
  echo "dir ${TRIMMED} doesn't exist"
fi
