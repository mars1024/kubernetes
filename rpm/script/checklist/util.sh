#!/bin/bash
#================================================================
# ScriptName: util.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018/12/26 18:08
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2018/12/26 18:08
# Function: unit function
#================================================================

get_runtime(){
    rpm -q pouch-container >/dev/null 2>&1
    if [[ $? -eq 0 ]]; then
        ret=$(/usr/local/bin/pouch info >/dev/null 2>&1)
        if [[ $? -ne 0 ]]; then
            echo "pouch daemon is offline"
            exit 1
        else
            echo "/usr/local/bin/pouch"
        fi
    else
        ret=$( /usr/bin/docker info >/dev/null 2>&1)
        if [[ $? -ne 0 ]]
        then
            echo "docker daemon is offline"
            exit 1
        else
            echo "/usr/bin/docker"
        fi
    fi
}

check_usage() {
    echo "Usage:  {start|stop}"
    exit 2 # bad usage
}

check_role() {
    if [[ "$UID" -ne 0 ]]; then
        echo "please run as root"
        exit 3 # bad user
    fi
}
