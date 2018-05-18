#!/bin/bash
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi

DATE=$1

WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# get env variables
source .env

NODEFINDER_NAME="geth-node-finder"
PROCESSED="${ARCHIVE_DIR}/${NODEFINDER_NAME}/processed-logs/${DATE}"
TMP="${PROCESSED}/tmp"
if cd ${PROCESSED} ; then
  # mainnet
  TYPE="mainnet"
  rm ${PROCESSED}/combined-${TYPE}-node-info.txt
  for i in `seq 30 -1 1`;
  do
    NUM_INSTANCE=${i}
    grep -Ff ${TYPE}-id-abbr.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}.txt
    grep 'inbound' ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt
    awk -F'|' '{print $3" "$4" inbound "$2}' ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt.tmp
    sort -t' ' -Vk1,1 -k4n,4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt.tmp | sort -t' ' -uk1,1 > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-first.txt
    sort -t' ' -Vk1,1 -k4nr,4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt.tmp | sort -t' ' -uk1,1 > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-last.txt
    join -a 1 -a 2 -o 0,2.2,1.3,1.4,2.4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-first.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-last.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-first-last.txt
    rm ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt* &

    grep -v 'inbound' ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt
    awk -F'|' '{print $3" "$4" outbound "$2}' ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt.tmp
    sort -t' ' -Vk1,1 -k4n,4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt.tmp | sort -t' ' -uk1,1 > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-first.txt
    sort -t' ' -Vk1,1 -k4nr,4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt.tmp | sort -t' ' -uk1,1 > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-last.txt
    join -a 1 -a 2 -o 0,2.2,1.3,1.4,2.4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-first.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-last.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-first-last.txt
    rm ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt* &

    join -a 1 -a 2 -o 0,1.2,1.3,1.4,1.5,2.2,2.3,2.4,2.5 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-first-last.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-first-last.txt | sed -E 's/[ ]+/|/g;s/\|$//' > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-first-last.txt
    cd ${WORKING_DIR} && python3 finalize-node-info.py ip-asn-db.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-first-last.txt ${PROCESSED}/${NUM_INSTANCE}-instance-${TYPE}-node-info.txt
    awk -F'|' -v instance="${NUM_INSTANCE}" '{print instance"|"$0}' ${PROCESSED}/${NUM_INSTANCE}-instance-${TYPE}-node-info.txt >> ${PROCESSED}/combined-${TYPE}-node-info.txt
  done

  # ethereum
  #TYPE="ethereum"
  #rm ${PROCESSED}/combined-${TYPE}-node-info.txt
  #for i in `seq 30 -1 1`;
  #do
  #  NUM_INSTANCE=${i}
  #  grep -Ff ${TYPE}-id-abbr.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}.txt
  #  grep 'inbound' ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt
  #  awk -F'|' '{print $3" "$4" inbound "$2}' ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt.tmp
  #  sort -t' ' -Vk1,1 -k4n,4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt.tmp | sort -t' ' -uk1,1 > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-first.txt
  #  sort -t' ' -Vk1,1 -k4nr,4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt.tmp | sort -t' ' -uk1,1 > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-last.txt
  #  join -a 1 -a 2 -o 0,2.2,1.3,1.4,2.4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-first.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-last.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-first-last.txt
  #  rm ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt* &

  #  grep -v 'inbound' ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt
  #  awk -F'|' '{print $3" "$4" outbound "$2}' ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt.tmp
  #  sort -t' ' -Vk1,1 -k4n,4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt.tmp | sort -t' ' -uk1,1 > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-first.txt
  #  sort -t' ' -Vk1,1 -k4nr,4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt.tmp | sort -t' ' -uk1,1 > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-last.txt
  #  join -a 1 -a 2 -o 0,2.2,1.3,1.4,2.4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-first.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-last.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-first-last.txt
  #  rm ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt* &

  #  join -a 1 -a 2 -o 0,1.2,1.3,1.4,1.5,2.2,2.3,2.4,2.5 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-first-last.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-first-last.txt | sed -E 's/[ ]+/|/g;s/\|$//' > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-first-last.txt
  #  cd ${WORKING_DIR} && python3 finalize-node-info.py ip-asn-db.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-first-last.txt ${PROCESSED}/${NUM_INSTANCE}-instance-${TYPE}-node-info.txt
  #  awk -F'|' -v instance="${NUM_INSTANCE}" '{print instance"|"$0}' ${PROCESSED}/${NUM_INSTANCE}-instance-${TYPE}-node-info.txt >> ${PROCESSED}/combined-${TYPE}-node-info.txt
  #done

  # devp2p
  TYPE="devp2p"
  rm ${PROCESSED}/combined-${TYPE}-node-info.txt
  for i in `seq 30 -1 1`;
  do
    NUM_INSTANCE=${i}
    grep 'inbound' ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt
    awk -F'|' '{print $3" "$4" inbound "$2}' ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt.tmp
    sort -t' ' -Vk1,1 -k4n,4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt.tmp | sort -t' ' -uk1,1 > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-first.txt
    sort -t' ' -Vk1,1 -k4nr,4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt.tmp | sort -t' ' -uk1,1 > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-last.txt
    join -a 1 -a 2 -o 0,2.2,1.3,1.4,2.4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-first.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-last.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-first-last.txt
    rm ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound.txt* &

    grep -v 'inbound' ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt
    awk -F'|' '{print $3" "$4" outbound "$2}' ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt.tmp
    sort -t' ' -Vk1,1 -k4n,4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt.tmp | sort -t' ' -uk1,1 > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-first.txt
    sort -t' ' -Vk1,1 -k4nr,4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt.tmp | sort -t' ' -uk1,1 > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-last.txt
    join -a 1 -a 2 -o 0,2.2,1.3,1.4,2.4 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-first.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-last.txt > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-first-last.txt
    rm ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound.txt* &

    join -a 1 -a 2 -o 0,1.2,1.3,1.4,1.5,2.2,2.3,2.4,2.5 ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-inbound-first-last.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-outbound-first-last.txt | sed -E 's/[ ]+/|/g;s/\|$//' > ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-first-last.txt
    cd ${WORKING_DIR} && python3 finalize-node-info.py ip-asn-db.txt ${TMP}/${NUM_INSTANCE}-instance-hello-disc-proto-${TYPE}-first-last.txt ${PROCESSED}/${NUM_INSTANCE}-instance-${TYPE}-node-info.txt
    awk -F'|' -v instance="${NUM_INSTANCE}" '{print instance"|"$0}' ${PROCESSED}/${NUM_INSTANCE}-instance-${TYPE}-node-info.txt >> ${PROCESSED}/combined-${TYPE}-node-info.txt
  done
else
  echo "dir ${PROCESSED} doesn't exist"
fi
