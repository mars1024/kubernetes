#!/bin/bash
#================================================================
# ScriptName: check runtime event
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018/12/26 14:43
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2018/12/26 14:43
# Function: analysis container runtime event
#================================================================
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source ${DIR}/../check-result-dispose.sh
source ${DIR}/../check-conf.sh
source ${DIR}/../util.sh

ACTION=$1
check_role

check_runtime_event_start(){
    echo "start check runtime event"

    event_file=${sigma_slave_check_dir}/runtime_event
    runtime=$(get_runtime)

    START=$(date +%s)

    # different os have different time format, for local debug
    os=$(uname -s)
    if [[ "$os" == "Linux" ]]; then
        END=$(date +%s  -d "+${check_time} second")
    elif [[ "$os" == "Darwin" ]]; then
        END=$(date -v+"${check_time}"S +%s)
    else
        echo "unknown OS"
        exit 1
    fi

    # watch runtime event and out put to event_file
    nohup "${runtime}" events --since="${START}" --until="${END}"   > "${event_file}" 2>&1 &

    # if no error, loop ${CHECK_TIME} second
    while [[ $(( $(date +%s) - START )) -lt ${check_time} ]]
    do

        # grep events which are we care
        events=$(< "${event_file}"  grep -E 'container die|container oom|container restart|container stop|container kill|container exec_die|network disconnect|network destroy|volume destroy' \
        | awk -F '(' '{print $1}' | awk '{ $1=""; print $0 }' | sort|uniq)

        if [[ -n "${events}" ]]; then
              stop_all_check_and_sigma_slave "runtime event :\n ${events}"
        fi

        date
        sleep ${interval}
    done
    echo "run time event ${check_success}" >> ${sigma_slave_check_dir}/${check_result_file}
    echo "check runtime event done, every thing is ok"
}

check_runtime_event_stop(){
    echo "shut down check run time event"
    ps x | grep check-runtime-event.sh | grep -v grep|  awk '{print $1}' | xargs kill -9
}

case "$ACTION" in
    start)
        check_runtime_event_start
    ;;
    stop)
        check_runtime_event_stop
    ;;
    *)
        check_usage
    ;;
esac