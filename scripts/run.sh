#!/bin/bash
# This script assumes required docker images are already built.
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: sudo ./run.sh num-instance"
  exit 1
fi

n=$(( $1 - 1 ))
WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
MYSQL_NAME="ethnodes-mysql"
NODEFINDER_NAME="geth-node-finder"

MYSQL_IMAGE="mysql:5.7-ethnodes"

# get env variables
source .env

# run mysql container
MYSQL_PORT=3306
echo "starting ${MYSQL_NAME} container..."
MYSQL_DIR="${ROOT_DIR}/ethnodes"
BACKUP_DIR="${ROOT_DIR}/ethnodes-backup"
mkdir -p -m 755 ${MYSQL_DIR} ${BACKUP_DIR}
docker run -dit --restart=always -p ${MYSQL_PORT}:3306 -h ${MYSQL_NAME} --name ${MYSQL_NAME} \
  --env MYSQL_DATABASE=${MYSQL_DB} \
  --env MYSQL_ROOT_PASSWORD=${MYSQL_PASSWORD} \
  --env MYSQL_USER=${MYSQL_USERNAME} \
  --env MYSQL_PASSWORD=${MYSQL_PASSWORD} \
  -v ${WORKING_DIR}/configs:/etc/mysql/conf.d \
  -v ${MYSQL_DIR}:/var/lib/mysql \
  -v ${BACKUP_DIR}:/backup \
  ${MYSQL_IMAGE}
echo "${MYSQL_NAME} started"

sleep 10

NODEFINDER_IMAGE="geth:node-finder"

# run node-finders
URL="research-scan.sprai.org"
NODEFINDER_PORT=1024
DATADIR="/root/.ethereum"
echo "starting ${NODEFINDER_NAME} containers..."
for i in `seq 0 ${n}`;
do
  IDENTITY="uiuc-${i}(${URL})"
  NODEFINDER_DIR="${ROOT_DIR}/${NODEFINDER_NAME}/${i}"
  [ -d "${NODEFINDER_DIR}" ] || mkdir -p -m 755 ${NODEFINDER_DIR}/${NODEFINDER_NAME}
  MYSQL_URL="${MYSQL_USERNAME}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DB}"
  PORT=$(( ${NODEFINDER_PORT}+${i} ))
  CMD="geth \
    --identity \"${IDENTITY}\" \
    --datadir \"${DATADIR}\" \
    --port ${PORT} \
    --verbosity 5 \
    --mysql \"${MYSQL_URL}\" \
    --logtofile \
    --redialfreq 1800 \
    --redialcheckfreq 5 \
    --redialexp 24 \
    --lastactive 1 \
    --maxnumfile 20480 \
    --maxredial 1000 \
    --pushfreq 1 \
    --maxsqlchunk 50 \
    --maxsqlqueue 1000000"
  docker run -dit --restart=always -h ${NODEFINDER_NAME}-${i} --name ${NODEFINDER_NAME}-${i} --net host -v ${NODEFINDER_DIR}:${DATADIR} -e CMD="${CMD}" --entrypoint '/bin/sh' ${NODEFINDER_IMAGE} -c "${CMD}"
 echo "${NODEFINDER_NAME}-${i} started"
done
