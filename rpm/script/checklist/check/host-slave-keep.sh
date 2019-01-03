#!/bin/bash
#****************************************************************#
# ScriptName: host-slave-keep.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018-12-24 17:21:55
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2018-12-24 17:22:04
# Function: shut down or open host-slave keep alive about sigma-slave
#***************************************************************#
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source ${DIR}/../check-conf.sh
source ${DIR}/../util.sh

ACTION=$1
check_role

host_slave_keep_start(){
    echo "start host-slave keep alive about sigma-slave"
    if [[ -e ${tian_ji_start_file} ]]
    then
       chmod 775 ${tian_ji_start_file}
    else
       echo "${tian_ji_start_file} not exist"
    fi
}

host_slave_keep_stop(){
    echo "shut down host-slave keep alive about sigma-slave"
    if [[ -e ${tian_ji_start_file} ]]
    then
       chmod 400 ${tian_ji_start_file}
    else
       echo "${tian_ji_start_file} not exist"
    fi
}

case "$ACTION" in
    start)
        host_slave_keep_start
    ;;
    stop)
        host_slave_keep_stop
    ;;
    *)
        check_usage
    ;;
esac