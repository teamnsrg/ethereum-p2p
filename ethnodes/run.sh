#!/bin/bash
# check if root
if [ "$EUID" -ne 0 ]; then
  echo "please run as root"
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "argument missing"
  echo "usage: sudo ./run.sh scale-size"
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

# run mysql containers
MYSQL_PORT=3306
echo "starting ${MYSQL_NAME} container..."
MYSQL_DIR="${ROOT_DIR}/ethnodes"
BACKUP_DIR="${ROOT_DIR}/ethnodes-backup"
mkdir -p -m 777 ${BACKUP_DIR}
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

read -p "Press any key to continue... "

# make node-finder container image
cd ${WORKING_DIR}/..
NODEFINDER_IMAGE="geth:node-finder"
docker build -t ${NODEFINDER_IMAGE} .

# run node-finders
URL="research-scan.sprai.org"
NODEFINDER_PORT=30310
DATADIR="/root/.ethereum"
echo "starting ${NODEFINDER_NAME} containers..."
for i in `seq 0 ${n}`;
do
  IDENTITY="uiuc-${i}(${URL})"
  NODEFINDER_DIR="${ROOT_DIR}/${i}"
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
    --maxnumfile 20480 \
    --pushfreq 1"
  docker run -d --restart=always -h ${NODEFINDER_NAME}-${i} --name ${NODEFINDER_NAME}-${i} --net host -v ${NODEFINDER_DIR}:${DATADIR} -e CMD="${CMD}" --entrypoint '/bin/sh' ${NODEFINDER_IMAGE} -c "${CMD}"
 echo "${NODEFINDER_NAME}-${i} started"
done
