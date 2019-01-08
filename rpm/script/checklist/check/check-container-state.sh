#!/bin/bash
#================================================================
# ScriptName: check-container-state.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018/12/26 20:57
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2018/12/26 20:57
# Function: check whether container num and state change  before and after rpm update (only running container)
#================================================================
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source ${DIR}/../check-result-dispose.sh
source ${DIR}/../check-conf.sh
source ${DIR}/../util.sh

ACTION=$1
check_role

check_container_state_start(){
    echo "start  check container state"

    # check file which is storage container state whether  exist
    container_state_file=${sigma_slave_check_dir}/container-state
    if [[ ! -f "${container_state_file}" ]];
    then
       echo "container state file not exist"
       echo "container state ${check_success}" >> ${sigma_slave_check_dir}/${check_result_file}
       return
    fi

    # if no error, loop ${CHECK_TIME} second
    START=$(date +%s)
    while [[ $(( $(date +%s) - START )) -lt ${check_time} ]]
    do
        runtime=$(get_runtime)

        # get new container state
        container_state_data=$( ${runtime} inspect  -f  "name={{.Name}};status={{.State.Status}};running={{.State.Running}};paused={{.State.Paused}};restarting={{.State.Restarting}};oomKilled={{.State.OOMKilled}};dead={{.State.Dead}};pid={{.State.Pid}};exitCode={{.State.ExitCode}};startedAt={{.State.StartedAt}};finishedAt={{.State.FinishedAt}}" `${runtime} ps -q`)

        # check whether container  num change
        before_count=$(< "${container_state_file}"  awk 'NR>1' | wc -l)
        after_count=$(echo "${container_state_data}" | wc -l)
        if [[ "${before_count}" -ne "${after_count}" ]]; then
             stop_all_check_and_sigma_slave "runtime event :\n
              container count change, before upgrade is ${before_count}, after upgrade is ${after_count}"
        fi

        # check every container
        for container in ${container_state_data[@]} ;
        do
         echo "container is: ${container}"

         container_name=$(echo "${container}" | awk -F ";" '{print $1}')
         echo "container name is: ${container_name}"

         new_container_state=$(echo "${container}" | awk -F ";" '{ $1=""; print $0 }')
         echo "container state is : ${new_container_state}"

         old_container_state=$(cat "${container_state_file}" | grep ${container_name} | awk -F ";" '{ $1=""; print $0 }')
         echo "old container state is ${old_container_state}"

         # if old container state not exist, which means this is a new container
         if [[ -z "${old_container_state}" ]]; then
              stop_all_check_and_sigma_slave "runtime event :\n
              container ${container_name} not exist in before update"
         fi

         # check container every state include Status, Running, restarting and so on
         new_container_state_array=(${new_container_state})
         old_container_state_array=(${old_container_state} )
         i=0
         while [[ ${i} -lt ${#new_container_state_array[@]} ]]
         do
            echo "new value is:${new_container_state_array[$i]}; old value is:::${old_container_state_array[$i]} "
            if [[ ! "${new_container_state_array[$i]}" == "${old_container_state_array[$i]}" ]]; then
                      stop_all_check_and_sigma_slave "runtime event :\n
                            container ${container_name} state not equal after upgrade, before upgrade is ${old_container_state_array[$i]},
                            after upgrade is ${new_container_state_array[$i]}"
            fi
            ((i++))
         done

       done
       date
       sleep  ${interval}
    done
    echo "container state ${check_success}" >> ${sigma_slave_check_dir}/${check_result_file}
    echo "check sigma-slave state done, every thing is ok"
}

check_container_state_stop(){
    echo "shut down check container state"
    ps x | grep check-container-state.sh | grep -v grep|  awk '{print $1}' | xargs kill -9
}

case "$ACTION" in
    start)
        check_container_state_start
    ;;
    stop)
        check_container_state_stop
    ;;
    *)
        check_usage
    ;;
esac
