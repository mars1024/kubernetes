#!/bin/bash
#================================================================
# ScriptName: check-conf.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018/12/25 19:43
# Modify Author: yaowei.cyw@alibaba-inc.com
# Modify Date: 2018/12/25 19:43
# Function: global check  conf
#================================================================
# sigma-slave process name, used to analysis the resources used by process
process_name="usr/local/bin/sigma-slave"
#process_name="baidu"

# the dir used to storage data before rpm upgrade
sigma_slave_check_dir="/etc/kubernetes/checkdata"
#sigma_slave_check_dir="/Users/changyaowei/work/src/k8s.io/kubernetes/rpm/script/checklist/testdata"

# the file which contains sigam-slave start parameter, through it we can get sigma-slave log directory
sigma_slave_start_conf="/etc/systemd/system/sigma-slave.service.d/sigma-slave-start.conf"

# tian ji start file which we can use to stop keep alive about sigma-slave
tian_ji_start_file="/cloud/app/sigma-slave/SigmaSlave#/sigma_slave/current/start"

# the time we run check shell
check_time=60

# the multiple of increase before we alert, if (now - before)/before > resources_increase,
resources_increase=1

# if resources increase of resources_increase more than resources increase times, we alert
resources_increase_times=2

# the interval we run check shell
interval=10

# set push_message to 0, not push message by dingTalk; when push_message is 1(default), push message by dingTalk when error.
push_message=1

# set check_debug to 0, stop sigma-slave and stop tian_ji_start_file ;
# when check_debug is 1(default), not kill sigma-slave and not stop tian_ji_start_file when error.
check_debug=1

# system inotify conf
inotify_max_queued_events=32768
inotify_max_user_instances=8192
inotify_max_user_watches=524288

# dingTalk message receiver
message_receiver=${message_receiver:-18268812036}
ding_token=${ding_token:-1a7310b61c973a4129b203231c998654177cdc183c94db5f53f12d1242fd31db}

# check result file, used to get check result
check_result_file="check-result-file"

# check fail flag
check_fail="check fail"

# check success flag, you should add this to result file, when check item success.
check_success="check success"

# check item num, if you add new check, should adjust this num
check_item=5

