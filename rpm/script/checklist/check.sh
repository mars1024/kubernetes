#!/bin/bash
#================================================================
# ScriptName: check.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018/12/27 16:40
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2018/12/27 16:40
# Function: start or stop all check
#================================================================
source ./util.sh
source ./check-conf.sh

ACTION=$1
check_role

check_start(){
   # start all check shell, if add new check, should start here
   echo "start check"
   echo "" > ${sigma_slave_check_dir}/${check_result_file}
   nohup ./check/host-slave-keep.sh start >> /tmp/sigma-slave-error.log 2>&1 &
   nohup ./check/check-container-exec.sh start >> /tmp/sigma-slave-error.log 2>&1 &
   nohup ./check/check-container-state.sh start >> /tmp/sigma-slave-error.log 2>&1 &
   nohup ./check/check-fatal-log.sh start >> /tmp/sigma-slave-error.log 2>&1 &
   nohup ./check/check-runtime-event.sh start >> /tmp/sigma-slave-error.log 2>&1 &
   nohup ./check/check-sigma-slave-process.sh start >> /tmp/sigma-slave-error.log 2>&1 &
   echo "start check done"
}

check_stop(){
    # stop all check shell, if add new check, should shop here
    echo "shut down check"
    nohup ./check/host-slave-keep.sh stop &
    nohup ./check/check-container-exec.sh stop &
    nohup ./check/check-container-state.sh stop &
    nohup ./check/check-fatal-log.sh stop &
    nohup ./check/check-runtime-event.sh stop &
    nohup ./check/check-sigma-slave-process.sh stop &
    echo "shut down check done"
}

check_status(){
    result_file=${sigma_slave_check_dir}/${check_result_file}
    if [[ ! -f "${result_file}" ]]; then
        echo "check result file not exist"
        return 1
    fi

    # if contains fail message,it  means check fail
    fail_message=$(grep "${check_fail}" "${result_file}")
    if [[ -n "${fail_message}" ]]; then
        echo "check fail"
        return 1
    fi

    # if success item num equal check item, it means check success
    success_item=$(grep "${check_success}" "${result_file}" | wc -l)
    if [[ ${success_item} -eq ${check_item} ]]; then
        echo "check success"
        return 0
    fi

    # in this case, it means check not end
    echo "check not end, should check ${check_item}, but only done ${success_item}"
    return 1
}

case "$ACTION" in
    start)
        check_start
    ;;
    stop)
        check_stop
    ;;
    status)
        check_status
    ;;
    *)
        check_usage
    ;;
esac