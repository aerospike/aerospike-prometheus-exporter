#! /usr/bin/env bash
# set -x

function join_by { local IFS="$1"; shift; echo "$*"; }

HOWMANY=$#
if [ ${HOWMANY} -lt 3 ] 
then
    echo "Usage: runstack.sh <prometheus.yml template> <local_ip> <aerospike_host:port> [<aerospike_host:port> ...]" 

    exit
fi

PROM_TEMPLATE=$(readlink -f $1)
shift

cd $(dirname $0)
aeromon=./aeromon
PROM_CFG_LOCATION=./docker/prometheus/prometheus.yml
HOST=$1
shift
FROM=49152
TO=65535


if [[ ${HOST} =~ ":" ]] 
then
    echo "${HOST} looks like it has a port. you must provide local ip accessible by docker as first argument. Hint probably in"
    hostname -I
    exit
fi

#https://unix.stackexchange.com/questions/55913/whats-the-easiest-way-to-find-an-unused-local-port
PORT_STR=$(comm -23 <(seq "$FROM" "$TO" | sort) <(ss -tan | awk '{print $4}' | cut -d':' -f2 | grep '[0-9]\{1,5\}' | sort) | head -n "$HOWMANY")
RANDOM_PORTS=($PORT_STR)

echo $PORT_STR

# start up all of the aeromons and track host/ports for prometheus conf
ii=0
TARGETS_ARR=()
for asd in $@
do
    ASD_HOST=${asd%:*}
    ASD_PORT=${asd#*:}
    RANDOM_PORT=${RANDOM_PORTS[$ii]}
#    echo "host ${HOST} port ${PORT} aero=$aeromon ${RANDOM_PORT} $ii"
    ${aeromon} -h ${ASD_HOST} -p ${ASD_PORT} -b ":${RANDOM_PORT}" &
    TARGETS_ARR+=("'$HOST:${RANDOM_PORT}'")
    ((ii++))
done

export TARGETS="[$(join_by , ${TARGETS_ARR[@]})]"
envsubst < ${PROM_TEMPLATE} > ${PROM_CFG_LOCATION};

# also kill all of the aeromons if this process dies
trap "trap - SIGTERM && kill -- -$$" SIGINT SIGTERM EXIT

docker-compose down -v && docker-compose rm -f && docker-compose pull && docker-compose up --build --force-recreate
