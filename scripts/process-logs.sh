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
  echo "usage: ./process-logs.sh type"
  exit 1
fi

i=$1

WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# get env variables
source .env

START="1524027553"
NODEFINDER_NAME="geth-node-finder"
PROCESSED="${ARCHIVE_DIR}/${NODEFINDER_NAME}/processed-logs"
if cd ${PROCESSED} ; then
  grep -Ff ethereum-id.txt ${i}-instance-hello-disc-proto.txt >${i}-instance-hello-disc-proto-ethereum.txt
  grep -Ff mainnet-id.txt ${i}-instance-hello-disc-proto.txt >${i}-instance-hello-disc-proto-mainnet.txt
  sort -t'|' -uVk2,2 ${i}-instance-hello-disc-proto.txt | awk -v startts="${START}" -F'|' '{printf("%f\n", $1-startts)}' | sort -V > ${i}-instance-ts.txt
  awk '{printf("%d,%d\n", NR-1, $1/3600)}' ${i}-instance-ts.txt | sort -t, -unk2 > ${i}-instance-ts-by-hr.txt
  sort -t'|' -uVk2,2 ${i}-instance-hello-disc-proto-ethereum.txt | awk -v startts="${START}" -F'|' '{printf("%f\n", $1-startts)}' | sort -V > ${i}-instance-ethereum-ts.txt
  awk '{printf("%d,%d\n", NR-1, $1/3600)}' ${i}-instance-ethereum-ts.txt | sort -t, -unk2 > ${i}-instance-ethereum-ts-by-hr.txt
  sort -t'|' -uVk2,2 ${i}-instance-hello-disc-proto-mainnet.txt | awk -v startts="${START}" -F'|' '{printf("%f\n", $1-startts)}' | sort -V > ${i}-instance-mainnet-ts.txt
  awk '{printf("%d,%d\n", NR-1, $1/3600)}' ${i}-instance-mainnet-ts.txt | sort -t, -unk2 > ${i}-instance-mainnet-ts-by-hr.txt
  if [ "${i}" == "30" ]; then
    grep 'inbound' ${i}-instance-hello-disc-proto.txt >${i}-instance-hello-disc-proto-inbound.txt
    sort -t'|' -uVk2,2 ${i}-instance-hello-disc-proto-inbound.txt | awk -v startts="${START}" -F'|' '{printf("%f\n", $1-startts)}' | sort -V > ${i}-instance-inbound-ts.txt
    grep -v 'inbound' ${i}-instance-hello-disc-proto.txt >${i}-instance-hello-disc-proto-outbound.txt
    sort -t'|' -uVk2,2 ${i}-instance-hello-disc-proto-outbound.txt | awk -v startts="${START}" -F'|' '{printf("%f\n", $1-startts)}' | sort -V > ${i}-instance-outbound-ts.txt
    grep 'inbound' ${i}-instance-hello-disc-proto-ethereum.txt >${i}-instance-hello-disc-proto-ethereum-inbound.txt
    sort -t'|' -uVk2,2 ${i}-instance-hello-disc-proto-ethereum-inbound.txt | awk -v startts="${START}" -F'|' '{printf("%f\n", $1-startts)}' | sort -V > ${i}-instance-ethereum-inbound-ts.txt
    grep -v 'inbound' ${i}-instance-hello-disc-proto-ethereum.txt >${i}-instance-hello-disc-proto-ethereum-outbound.txt
    sort -t'|' -uVk2,2 ${i}-instance-hello-disc-proto-ethereum-outbound.txt | awk -v startts="${START}" -F'|' '{printf("%f\n", $1-startts)}' | sort -V > ${i}-instance-ethereum-outbound-ts.txt
    grep 'inbound' ${i}-instance-hello-disc-proto-mainnet.txt >${i}-instance-hello-disc-proto-mainnet-inbound.txt
    sort -t'|' -uVk2,2 ${i}-instance-hello-disc-proto-mainnet-inbound.txt | awk -v startts="${START}" -F'|' '{printf("%f\n", $1-startts)}' | sort -V > ${i}-instance-mainnet-inbound-ts.txt
    grep -v 'inbound' ${i}-instance-hello-disc-proto-mainnet.txt >${i}-instance-hello-disc-proto-mainnet-outbound.txt
    sort -t'|' -uVk2,2 ${i}-instance-hello-disc-proto-mainnet-outbound.txt | awk -v startts="${START}" -F'|' '{printf("%f\n", $1-startts)}' | sort -V > ${i}-instance-mainnet-outbound-ts.txt
  fi
else
  echo "dir ${PROCESSED} doesn't exist"
fi
