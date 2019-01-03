#!/bin/bash
#================================================================
# ScriptName: check-host-env.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018/12/27 16:02
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2018/12/27 16:02
# Function: before rpm install check host env
#================================================================
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source ${DIR}/../check-result-dispose.sh
source ${DIR}/../check-conf.sh

check_runtime(){
    rpm -q pouch-container
    if [[ $? -eq 0 ]]; then
        ret=$(/usr/local/bin/pouch info)
        if [[ $? -ne 0 ]]; then
           echo "pouch daemon is offline"
           if [[ ${push_message} -eq 1 ]]; then
                message_notify "pouch daemon is offline, not install sigmalet"
           fi
           exit 1
        fi
    else
        ret=$(/usr/bin/docker info)
        if [[ $? -ne 0 ]]; then
           echo "docker daemon is offline"
           if [[ ${push_message} -eq 1 ]]; then
                message_notify "docker daemon is offline, not install sigmalet"
           fi
           exit 1
        fi
    fi
}

check_inotify(){
    max_queued_events=$(sysctl -n fs.inotify.max_queued_events)
    max_user_instances=$(sysctl -n fs.inotify.max_user_instances)
    max_user_watches=$(sysctl -n fs.inotify.max_user_watches)
    
    if [[ ${max_queued_events} -lt ${inotify_max_queued_events} || ${max_user_instances} -lt ${inotify_max_user_instances} || ${max_user_watches} -lt ${inotify_max_user_watches} ]]; then
        error_msg="\n fs.inotify.max_queued_events value is ${max_queued_events}, should bigger than  ${inotify_max_queued_events} \n
          fs.inotify.max_user_instances value is ${max_user_instances}, should bigger than  ${inotify_max_user_instances}\n
          fs.inotify.max_user_watches value is ${max_user_watches}, should bigger than  ${inotify_max_user_watches}"

        if [[ ${push_message} -eq 1 ]]; then
            message_notify ${error_msg}
        fi
        echo ${error_msg}
        exit 1
    fi

}

check_runtime
check_inotify