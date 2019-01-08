#!/bin/bash
#****************************************************************#
# ScriptName: check-result-dispose.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018-12-24 16:43:06
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2018-12-24 16:43:06
# Function: dispose check result
# product: 1a7310b61c973a4129b203231c998654177cdc183c94db5f53f12d1242fd31db
# test: 7b1de0aa48a65256988fa70d059e6b88e65d2b2dfaa1344fcb917983654d7a1c
#***************************************************************#
source ./check-conf.sh

message_notify() {
    # get host ip
    ip=$(/bin/bash -c 'hostname -i' 2>/dev/null)
    if [[ -z "$ip" ]]; then
          echo "can't get ip hostname -i cmd"
          ip=$(/bin/bash -c 'hostname')
    fi

    # get sigma-slave version
    sigma_slave_version=$(/usr/local/bin/sigma-slave --version 2>/dev/null)

    # replace '"' in message, to avoid message truncation
    info=$(echo $* | sed 's/"//g')

    # join together all message
    ct="host: [${ip}](https://sa.alibaba-inc.com/ops/terminal.html?source=sigma&ip=${ip}) upgrade sigmalet to version :  ${sigma_slave_version} error \n: ${info} "
    # send message by dingTalk
    curl 'https://oapi.dingtalk.com/robot/send?access_token=1a7310b61c973a4129b203231c998654177cdc183c94db5f53f12d1242fd31db' \
            -H 'Content-Type: application/json' \
            -d '{
                    "msgtype": "markdown",
                    "markdown": {
                        "title": "sigmalet check",
                        "text": "'"$ct"' \n @'${message_receiver}' \n "
                        },
                     "at": {
                             "atMobiles": [
                                 "'${message_receiver}'"
                             ],
                             "isAtAll": false
                         }
                }'
}

stop_all_check(){
    # push message or just echo
    if [[ ${push_message} -eq 1 ]]; then
        message_notify $*
    else
        echo "$*" >>  /tmp/sigma-slave-error.log 2>&1
    fi

    # stop all check shell
    sh ./check.sh stop
    echo ${check_fail} >> ${sigma_slave_check_dir}/${check_result_file}
}

stop_all_check_and_sigma_slave(){
    # stop all check
    stop_all_check $*

    # stop sigma-slave process
    if [[ ${check_debug} -eq 0 ]]; then
        systemctl stop sigma-slave
    fi
}
