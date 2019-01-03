#!/bin/bash
#================================================================
# ScriptName: check-sigma-slave-process.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018/12/25 19:28
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2018/12/25 19:28
# Function: check the resources used by sigma slave process
#================================================================
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source ${DIR}/../check-result-dispose.sh
source ${DIR}/../check-conf.sh
source ${DIR}/../util.sh

ACTION=$1
check_role

check_sigma_slave_process_start(){
    echo "start check sigma-slave process"
    #使用掉的 CPU 资源百分比
    cpu_before=0
    #所占用的物理内存百分比
    mem_before=0
    #使用掉的虚拟内存量 (Kbytes)
    vsz_before=0
    #占用的固定的内存量 (Kbytes)
    rss_before=0

    process_file=${sigma_slave_check_dir}/sigma-slave-process
    if [[ ! -f ${process_file} ]];
    then
        echo "sigma-slave process file not exist"
    else
        read cpu_before mem_before vsz_before rss_before < <(< "${process_file}"  awk 'NR==2' | awk '{print $1,$2,$3,$4}')
    fi

    echo "cpu_before is ${cpu_before},mem_before is ${mem_before},vsz_before is ${vsz_before},rss_before is ${rss_before}"

    cpu_increase_count=0
    mem_increase_count=0
    vsz_increase_count=0
    rss_increase_count=0

    # if no error, loop ${CHECK_TIME} second
    START=$(date +%s)
    while [[ $(( $(date +%s) - START )) -lt ${check_time} ]]
    do
        process_data=$(ps aux | grep ${process_name} | grep -v grep)
        if [[ $? -eq 1 ]];
        then
            # not stop sigma-slave process to keep the scene
            stop_all_check "sigma-slave process not exist"
        fi
        date > "${process_file}"
        echo "${process_data}" | awk '{print $3,$4,$5,$6}' >> "${process_file}"

        read cpu mem vsz rss < <(echo "${process_data}"  | awk '{print $3,$4,$5,$6}')
        echo "cpu is ${cpu},mem is ${mem},vsz is ${vsz},rss is ${rss}"

        # analysis cpu
        if [[ $(echo "${cpu_before} > 0" | bc -l) -eq  1 ]];
        then
            if [[ $(echo "(${cpu}-${cpu_before})/${cpu_before} > $resources_increase" | bc -l ) -eq 1 ]]; then
               ((cpu_increase_count))
               if [[ "${cpu_increase_count}" -ge "${resources_increase_times}" ]]; then
                    # not stop sigma-slave process to keep the scene
                    stop_all_check "sigma-slave process cpu increase more than $(echo "${resources_increase} * 100"| bc)%, ${cpu_increase_count} times"
               fi
            else
                cpu_increase_count=0
            fi
        fi

        # analysis mem
        if [[ $(echo "${mem_before} > 0" | bc -l) -eq  1 ]];
        then
            if [[ $(echo "(${mem}-${mem_before})/${mem_before} > $resources_increase" | bc -l ) -eq 1 ]]; then
               ((mem_increase_count++))
               if [[ "${mem_increase_count}" -ge "${resources_increase_times}" ]]; then
                    # not stop sigma-slave process to keep the scene
                    stop_all_check "sigma-slave process mem increase more than $(echo "${resources_increase} * 100"| bc)%, ${mem_increase_count} times"
               fi
            else
               mem_increase_count=0
            fi
        fi

        # analysis vsz
        if [[ $(echo "${vsz_before} > 0" | bc -l) -eq  1 ]];
        then
            if [[ $(echo "(${vsz}-${vsz_before})/${vsz_before} > $resources_increase" | bc -l ) -eq 1 ]]; then
               ((vsz_increase_count++))
               if [[ "${vsz_increase_count}" -ge "${resources_increase_times}" ]]; then
                    # not stop sigma-slave process to keep the scene
                    stop_all_check "sigma-slave process vsz increase more than  $(echo "${resources_increase} * 100"| bc)%, ${vsz_increase_count} times"
               fi
            else
               vsz_increase_count=0
            fi
        fi

        # analysis rss
        if [[ $(echo "${rss_before} > 0" | bc -l) -eq  1 ]];
        then
            if [[ $(echo "(${rss}-${rss_before})/${rss_before} > $resources_increase" | bc -l ) -eq 1 ]]; then
               ((rss_increase_count++))
               if [[ "${rss_increase_count}" -ge "${resources_increase_times}" ]]; then
                   # not stop sigma-slave process to keep the scene
                   stop_all_check "sigma-slave process rss increase more than  $(echo "${resources_increase} * 100"| bc)%, ${rss_increase_count} times"
               fi
            else
               rss_increase_count=0
            fi
        fi

        cpu_before=${cpu}
        mem_before=${mem}
        vsz_before=${vsz}
        rss_before=${rss}
        echo "cpu_before is ${cpu_before},mem_before is ${mem_before},vsz_before is ${vsz_before},rss_before is ${rss_before}"

        date
        sleep ${interval}
    done
    echo "sigma-slave process ${check_success}" >> ${sigma_slave_check_dir}/${check_result_file}
    echo "check sigma-slave process  done, every thing is ok"
}

check_sigma_slave_process_stop(){
    echo "shut down check sigma slave process"
    ps x | grep check-sigma-slave-process.sh | grep -v grep|  awk '{print $1}' | xargs kill -9
}

case "$ACTION" in
    start)
        check_sigma_slave_process_start
    ;;
    stop)
        check_sigma_slave_process_stop
    ;;
    *)
        check_usage
    ;;
esac
