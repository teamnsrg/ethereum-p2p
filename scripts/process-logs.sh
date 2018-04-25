#!/bin/bash
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -lt 1 ]; then
  echo "argument missing"
  echo "usage: ./process-logs.sh num-instance type"
  exit 1
fi

i=$1
TYPE=$2

WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# get env variables
source .env

START="1524027553"
NODEFINDER_NAME="geth-node-finder"
PROCESSED="${ARCHIVE_DIR}/${NODEFINDER_NAME}/processed-logs"
if cd ${PROCESSED} ; then
  if [ "${TYPE}" != "" ]; then
    grep -Ff ${TYPE}-id.txt ${i}-instance-hello-disc-proto.txt >${i}-instance-hello-disc-proto-${TYPE}.txt
    TYPE="-${TYPE}"
  fi
  sort -t'|' -uVk2,2 ${i}-instance-hello-disc-proto${TYPE}.txt > ${i}-instance-hello-disc-proto${TYPE}.uniq-nodeid
  awk -v startts="${START}" -F'|' '{printf("%f\n", $1-startts)}' ${i}-instance-hello-disc-proto${TYPE}.uniq-nodeid | sort -V > ${i}-instance${TYPE}-ts.txt
  awk '{printf("%d,%d\n", NR-1, $1/3600)}' ${i}-instance${TYPE}-ts.txt | sort -t, -unk2 > ${i}-instance${TYPE}-ts-by-hr.txt
  grep 'inbound' ${i}-instance-hello-disc-proto${TYPE}.txt >${i}-instance-hello-disc-proto${TYPE}-inbound.txt
  sort -t'|' -uVk2,2 ${i}-instance-hello-disc-proto${TYPE}-inbound.txt > ${i}-instance-hello-disc-proto${TYPE}-inbound.uniq-nodeid
  awk -v startts="${START}" -F'|' '{printf("%f\n", $1-startts)}' ${i}-instance-hello-disc-proto${TYPE}-inbound.uniq-nodeid | sort -V > ${i}-instance${TYPE}-inbound-ts.txt
  grep -v 'inbound' ${i}-instance-hello-disc-proto${TYPE}.txt >${i}-instance-hello-disc-proto${TYPE}-outbound.txt
  sort -t'|' -uVk2,2 ${i}-instance-hello-disc-proto${TYPE}-outbound.txt > ${i}-instance-hello-disc-proto${TYPE}-outbound.uniq-nodeid
  awk -v startts="${START}" -F'|' '{printf("%f\n", $1-startts)}' ${i}-instance-hello-disc-proto${TYPE}-outbound.uniq-nodeid | sort -V > ${i}-instance${TYPE}-outbound-ts.txt
  if [ "${TYPE}" == "-mainnet" ]; then
    awk -F'|' '{print $2"|"$3" inbound "$1}' ${i}-instance-hello-disc-proto${TYPE}-inbound.txt >${i}-instance-hello-disc-proto${TYPE}-inbound.txt.tmp
    sort -t' ' -Vk1,1 -k3n,3 ${i}-instance-hello-disc-proto${TYPE}-inbound.txt.tmp | sort -t' ' -uk1,1 > ${i}-instance-hello-disc-proto${TYPE}-inbound-first.txt
    sort -t' ' -Vk1,1 -k3nr,3 ${i}-instance-hello-disc-proto${TYPE}-inbound.txt.tmp | sort -t' ' -uk1,1 > ${i}-instance-hello-disc-proto${TYPE}-inbound-last.txt
    rm ${i}-instance-hello-disc-proto${TYPE}-inbound.txt.tmp
    join -a 1 -a 2 -o 0,1.2,1.3,2.3 ${i}-instance-hello-disc-proto${TYPE}-inbound-first.txt ${i}-instance-hello-disc-proto${TYPE}-inbound-last.txt > ${i}-instance-hello-disc-proto${TYPE}-inbound-first-last.txt
    awk -F'|' '{print $2"|"$3" inbound "$1}' ${i}-instance-hello-disc-proto${TYPE}-outbound.txt >${i}-instance-hello-disc-proto${TYPE}-outbound.txt.tmp
    sort -t' ' -Vk1,1 -k3n,3 ${i}-instance-hello-disc-proto${TYPE}-outbound.txt.tmp | sort -t' ' -uk1,1 > ${i}-instance-hello-disc-proto${TYPE}-outbound-first.txt
    sort -t' ' -Vk1,1 -k3nr,3 ${i}-instance-hello-disc-proto${TYPE}-outbound.txt.tmp | sort -t' ' -uk1,1 > ${i}-instance-hello-disc-proto${TYPE}-outbound-last.txt
    rm ${i}-instance-hello-disc-proto${TYPE}-outbound.txt.tmp
    join -a 1 -a 2 -o 0,1.2,1.3,2.3 ${i}-instance-hello-disc-proto${TYPE}-outbound-first.txt ${i}-instance-hello-disc-proto${TYPE}-outbound-last.txt > ${i}-instance-hello-disc-proto${TYPE}-outbound-first-last.txt
    join -a 1 -a 2 -o 0,1.2,1.3,1.4,2.2,2.3,2.4 ${i}-instance-hello-disc-proto${TYPE}-inbound-first-last.txt ${i}-instance-hello-disc-proto${TYPE}-outbound-first-last.txt | sed -E 's/[ ]+/|/g;s/\|$//' > ${i}-instance-hello-disc-proto${TYPE}-first-last.txt
    cd ${WORKING_DIR} && python3 finalize-node-info.py ip-asn-db.txt ${PROCESSED}/${i}-instance-hello-disc-proto${TYPE}-first-last.txt ${PROCESSED}/${i}-instance${TYPE}-node-info.txt
  fi
else
  echo "dir ${PROCESSED} doesn't exist"
fi
