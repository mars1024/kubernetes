#!/bin/bash
#****************************************************************#
# ScriptName: pre-rpm-install.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018-12-25 17:28:08
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2018-12-25 17:28:13
# Function: storage state, such as: container state, sigma-slave process state and so on
#***************************************************************#
source ./check-conf.sh
source ./util.sh

# storage resource which are used by sigma-slave
storage_sigma_slave_process() {
    mkdir -p ${sigma_slave_check_dir}
    if [[ $? -ne 0 ]]; then
        echo "make dir ${sigma_slave_check_dir} error"
        return
    fi

	process_file=${sigma_slave_check_dir}/sigma-slave-process
	process_data=$(ps aux | grep ${process_name} | grep -v grep | awk '{print $3,$4,$5,$6}')
	date > "${process_file}"
	echo "${process_data}" >> "${process_file}"
}

# storage all containers state include dead container
storage_container_state() {
    mkdir -p ${sigma_slave_check_dir}
    if [[ $? -ne 0 ]]; then
        echo "make dir ${sigma_slave_check_dir} error"
        return
    fi

	# only consider sigma 3.1 container, ignore 2.0 and pause container
	runtime=$(get_runtime)
	if [[ ${runtime} == *pouch* ]]; then
		label_filter="label=io.kubernetes.pouch.type=container"
	else
		label_filter="label=io.kubernetes.docker.type=container"
	fi
	container_state_file=${sigma_slave_check_dir}/container-state
	container_state_data=$( ${runtime} inspect  -f  "name={{.Name}};status={{.State.Status}};running={{.State.Running}};paused={{.State.Paused}};restarting={{.State.Restarting}};oomKilled={{.State.OOMKilled}};dead={{.State.Dead}};pid={{.State.Pid}};exitCode={{.State.ExitCode}};startedAt={{.State.StartedAt}};finishedAt={{.State.FinishedAt}}" `${runtime} ps -f "${label_filter}" -q` )

	date > "${container_state_file}"
	echo "${container_state_data}" >> "${container_state_file}"
}

# storage container id which we can execute exec cmd, only sigma3.1 container
storage_container_exec() {
    mkdir -p ${sigma_slave_check_dir}
    if [[ $? -ne 0 ]]; then
        echo "make dir ${sigma_slave_check_dir} error"
        return
    fi

	# only consider sigma 3.1 container, ignore 2.0 and pause container
	runtime=$(get_runtime)
	if [[ ${runtime} == *pouch* ]]; then
		label_filter="label=io.kubernetes.pouch.type=container"
	else
		label_filter="label=io.kubernetes.docker.type=container"
	fi
	container_exec_file=${sigma_slave_check_dir}/container-exec
	container_ids=$(${runtime} ps -f "${label_filter}" -q)

	echo "container ids ${container_ids}"
	date > "${container_exec_file}"

	for container_id in ${container_ids};
	do
		echo "container id is ${container_id}"
		ret=$(${runtime} exec "${container_id}" date)
		if [[ $? -eq 0 ]];
		then
			echo "${container_id}" >> "${container_exec_file}"
		fi
	done
}

storage_sigma_slave_process
storage_container_state
storage_container_exec
