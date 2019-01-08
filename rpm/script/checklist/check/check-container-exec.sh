#!/bin/bash
#================================================================
# ScriptName: check-container-exec.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018/12/27 13:39
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2018/12/27 13:39
# Function:  check containers which are we can execute exec
# before rpm update whether can execute exec
#================================================================
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source ${DIR}/../check-result-dispose.sh
source ${DIR}/../check-conf.sh
source ${DIR}/../util.sh

ACTION=$1
check_role

check_container_exec_start(){
    echo "start check container exec"
    # check file which is storage container ID whether exist
    container_exec_file=${sigma_slave_check_dir}/container-exec
    if [[ ! -f "${container_exec_file}" ]];
    then
       echo "container exec file not exist"
       echo "container exec ${check_success}" >> ${sigma_slave_check_dir}/${check_result_file}
       return
    fi

    # if no error, loop ${CHECK_TIME} second
    START=$(date +%s)
    while [[ $(( $(date +%s) - START )) -lt ${check_time} ]]
    do
        # get container id which means we can execute exec before update rpm from container-exec file.
        runtime=$(get_runtime)
        container_ids=$(< "${container_exec_file}" awk 'NR>1')
        echo "container ids ${container_ids}"

        for container_id in $container_ids;
	    do
		    echo "container id is ${container_id}"

            # check if we can execute exec cmd
		    ret=$(${runtime} exec "${container_id}" date)
		    if [[ $? -ne 0 ]];
		    then
			      stop_all_check_and_sigma_slave "exec event :\n
                            container ${container_id} can't exec after upgrade;"
		    fi
	    done

        date
        sleep ${interval}
    done
    echo "container exec ${check_success}" >> ${sigma_slave_check_dir}/${check_result_file}
    echo "check sigma-slave exec  done, every thing is ok"
}

check_container_exec_stop(){
    echo "shut down check container exec"
    ps x | grep check-container-exec.sh | grep -v grep|  awk '{print $1}' | xargs kill -9
}

case "$ACTION" in
    start)
        check_container_exec_start
    ;;
    stop)
        check_container_exec_stop
    ;;
    *)
        check_usage
    ;;
esac
