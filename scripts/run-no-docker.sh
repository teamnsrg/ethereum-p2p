#!/bin/bash
# This script assumes required docker images are already built.
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: sudo ./run-no-docker.sh num-instance"
  exit 1
fi

n=$(( $1 - 1 ))
WORKING_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
MYSQL_NAME="ethnodes-mysql"
NODEFINDER_NAME="geth-node-finder"

# build mysql container image
MYSQL_IMAGE="mysql:5.7-ethnodes"
docker build -t ${MYSQL_IMAGE} -f ethnodes-dockerfile .

# get env variables
source .env

# run mysql container
MYSQL_PORT=3306
echo "starting ${MYSQL_NAME} container..."
MYSQL_DIR="${ROOT_DIR}/ethnodes"
BACKUP_DIR="${ROOT_DIR}/ethnodes-backup"
mkdir -p -m 755 ${MYSQL_DIR} ${BACKUP_DIR}
docker run -d --restart=always -p ${MYSQL_PORT}:3306 -h ${MYSQL_NAME} --name ${MYSQL_NAME} \
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

cd ${WORKING_DIR}/..
make geth
cp ${WORKING_DIR}/../build/bin/geth /usr/bin/geth

cd ${WORKING_DIR}
DATADIR="${ROOT_DIR}/${NODEFINDER_NAME}"
mkdir -p -m 755 ${DATADIR}
# run node-finders
for i in `seq 0 ${n}`;
do
  ./node-finder-loop.sh ${i} >>${DATADIR}/${NODEFINDER_NAME}-${i}-loop.log 2>&1 &
  echo "${NODEFINDER_NAME}-${i} loop started"
done
