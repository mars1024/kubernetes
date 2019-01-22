#!/bin/bash
#================================================================
# ScriptName: check-cpu-manger-state.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2019/1/14 16:34
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2019/1/14 16:34
# Function: check cpu_manager_state file and move
#================================================================

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source ${DIR}/../check-result-dispose.sh
source ${DIR}/../check-conf.sh
source ${DIR}/../util.sh

ACTION=$1
check_role

function check_and_mv_file() {
    if [[ ! -f ${sigma_slave_start_conf} ]]; then
        return
    fi

    root_dir=$(grep root-dir ${sigma_slave_start_conf} |tr ' '  '\n' | grep root-dir | awk -F "=" '{print $2}')
    cpu_file=${root_dir}/cpu_manager_state
    if [[ ! -f ${cpu_file} ]]; then
        return
    fi

    policy=$(grep '"policyName":"none"' ${cpu_file})
    if [[ -n "${policy}"  ]]; then
        mv ${cpu_file} ${root_dir}/cpu_manager_state_none_bak
    fi
}
check_and_mv_file