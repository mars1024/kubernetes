#!/bin/bash
#****************************************************************#
# ScriptName: check-fatal-log.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018-12-24 18:10:32
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2018-12-24 18:10:38
# Function: check sigma-slave fatal log, find abnormal log
#***************************************************************#
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source ${DIR}/../check-result-dispose.sh
source ${DIR}/../check-conf.sh
source ${DIR}/../util.sh

ACTION=$1
check_role

# get log dir by sigma-slave conf file
get_log_dir(){
    if [[ -f ${sigma_slave_start_conf} ]]
    then
       log_dir=$(grep log-dir ${sigma_slave_start_conf} |tr ' '  '\n' | grep log-dir | awk -F "=" '{print $2}')
       echo "${log_dir}"
    else
       echo ""
    fi
}

check_fatal_log_start(){
    echo "start sigma-slave fatal log analysis"

    # check sigma-slave fatal log whether exist.
    log_dir=$(get_log_dir)
    echo "log dir is: $log_dir"

    # if no fatal, loop ${CHECK_TIME} second
    START=$(date +%s)
    while [[ $(( $(date +%s) - START )) -lt ${check_time} ]]
    do
        fatal_log_file="${log_dir}/sigma-slave.FATAL"
        info_log_file="${log_dir}/sigma-slave.INFO"
        if [[ -f "${fatal_log_file}" && -f "${info_log_file}" ]];
        then
            # log file name contain pid ,such as : sigma-slave.astro172029200017.root.log.INFO.20190102-192904.68299
            info_log_process_pid=$(readlink "${info_log_file}" | awk -F '.' '{print $NF}')
            fatal_log_process_pid=$(readlink "${fatal_log_file}" | awk -F '.' '{print $NF}')

            # fatal log should create by current process.
            if [[ "${info_log_process_pid}" == "${fatal_log_process_pid}" ]]; then
                  # if file contains abnormal log, we should send through dingTalk and end the analysis.
                  if [[ $(egrep -c "([0-1][0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9])\." ${fatal_log_file}) -gt 0 ]];
                  then
                    stop_all_check_and_sigma_slave "sigma-slave fatal log :$(egrep "([0-1][0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9])\." ${fatal_log_file})"
                  fi
            else
               echo "sigma-slave fatal log not create by current process"
            fi
        else
            echo "sigma-slave fatal log not exist"
        fi

        date
        sleep  ${interval}
    done
    echo "sigma-slave log  ${check_success}" >> ${sigma_slave_check_dir}/${check_result_file}
    echo "check fatal log done, every thing is ok"
}

check_fatal_log_stop(){
    echo "shut down check fatal log"
    ps x | grep check-fatal-log.sh | grep -v grep|  awk '{print $1}' | xargs kill -9
}

case "$ACTION" in
    start)
       check_fatal_log_start
    ;;
    stop)
       check_fatal_log_stop
    ;;
    *)
       check_usage
    ;;
esac
