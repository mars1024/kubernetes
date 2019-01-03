#!/bin/bash

os=`awk '{print $7}' /etc/redhat-release 2>/dev/null`
if [ "$os"x != "7.2"x ]; then
  echo "5u6u os skip"
  exit 0
fi

if [[ $# -lt 2 ]]; then
  echo "should include two param, First Parameter : kubelet service path, Second Parameter : kubelet log cron file"
  exit 1
fi

kubelet_service_path=$1
kubelet_log_cron_file=$2


# change root dir
change_root_dir(){
  kubelet_root_dir=${1%/*}"/kubernetes/lib/sigmaSlave"
  kubelet_root_dir=$(echo $kubelet_root_dir | sed 's#\/#\\\/#g')
  sed -i "s/--root-dir=\/home\/t4\/kubernetes\/lib\/sigmaSlave/--root-dir=$kubelet_root_dir/g" $kubelet_service_path
}

# change sigma-slave log dir
change_log_dir(){
  kubelet_log_dir=${1%/*}"/kubernetes/logs"
  # make sure log file exist
  mkdir -p $kubelet_log_dir
  kubelet_log_dir=$(echo $kubelet_log_dir | sed 's#\/#\\\/#g')
  sed -i "s/--log-dir=\/home\/t4\/kubernetes\/logs/--log-dir=$kubelet_log_dir/g" $kubelet_service_path
}

# change sigma-slave log cron conf
change_log_cron_file(){
  kubelet_log_dir=${1%/*}"/kubernetes/logs"
  kubelet_log_dir=$(echo $kubelet_log_dir | sed 's#\/#\\\/#g')
  sed -i "s/\/home\/t4\/kubernetes\/logs/$kubelet_log_dir/g" $kubelet_log_cron_file
}

# change seccomp profile root
change_seccomp_profile_root(){
  seccomp_profile_root=${1%/*}"/kubernetes/lib/sigmaSlave/seccomp"
  seccomp_profile_root=$(echo $seccomp_profile_root | sed 's#\/#\\\/#g')
  sed -i "s/--seccomp-profile-root=\/home\/t4\/kubernetes\/lib\/sigmaSlave\/seccomp/--seccomp-profile-root=$seccomp_profile_root/g" $kubelet_service_path
}

# change runtime cgroupfs
change_runtime_cgroupfs(){
    runtime_root_dir=$(echo $1 | sed 's#\/#\\\/#g')
    sed -i "s/--runtime-cgroups=\/home\/t4\/docker/--runtime-cgroups=$runtime_root_dir/g" $kubelet_service_path
}

# create cgroup directories for pai deployment
cgroup_config(){
  CGROUP_ROOT="sigma-be"

  if [ ! -d "/sys/fs/cgroup/cpu/$CGROUP_ROOT" ]; then
    mkdir /sys/fs/cgroup/cpu/$CGROUP_ROOT
  fi

  if [ ! -d "/sys/fs/cgroup/net_cls/$CGROUP_ROOT" ]; then
      mkdir /sys/fs/cgroup/net_cls/$CGROUP_ROOT
  fi

  if [ ! -d "/sys/fs/cgroup/memory/$CGROUP_ROOT" ]; then
      mkdir /sys/fs/cgroup/memory/$CGROUP_ROOT
  fi

  if [ ! -d "/sys/fs/cgroup/systemd/$CGROUP_ROOT" ]; then
      mkdir /sys/fs/cgroup/systemd/$CGROUP_ROOT
  fi

  echo 2 > /sys/fs/cgroup/cpu/$CGROUP_ROOT/cpu.shares
  echo 2000000 > /sys/fs/cgroup/cpu/$CGROUP_ROOT/cpu.cfs_quota_us
  #TDOO set memory
}

modify_start_conf_for_pai_in_et2(){
  site=$(/bin/bash -c hostinfo --cmdb 2>/dev/null | grep logicRegion | awk '{print $2}' |  sed 's/ //g' | tr 'A-Z' 'a-z')
  if [[ -z "$site" ]]; then
	  echo "can't get site, exit"
	  exit 0
  fi

  if [ $site != "et2" ]; then
    exit 0
  fi

  cgroup_config

  cpuNums=$(cat /proc/cpuinfo| grep "processor"| sort| uniq| wc -l)
  reservedMemByG=$(awk 'BEGIN{printf "%.1f\n",('$cpuNums'*2.2)}')

  # change all special args for pai.
  sed -i "s/--cluster-domain=cluster.local/--cluster-dns=11.140.98.87 --system-reserved=memory=${reservedMemByG}Gi --cgroup-root=\/sigma-be/g" $kubelet_service_path
}

rpm -q pouch-container

if [ $? -eq 0 ]; then
    echo "pouch daemon"
    runtime_root_dir=$(/usr/local/bin/pouch info 2>/dev/null | grep 'Pouch Root Dir:' | grep -v "#"  | awk -F: '{print $2}' | sed 's/ //g')
    change_root_dir $runtime_root_dir

    change_log_dir $runtime_root_dir

    change_log_cron_file  $runtime_root_dir

    change_seccomp_profile_root $runtime_root_dir

    change_runtime_cgroupfs $runtime_root_dir

    sed -i "s/--container-runtime=docker/--container-runtime=remote/g" $kubelet_service_path
    sed -i "s/container-runtime-endpoint=unix:\/\/\/var\/run\/dockershim.sock/container-runtime-endpoint=unix:\/\/\/var\/run\/pouchcri.sock/g" $kubelet_service_path

else
  rpm -qa | grep "alidocker-1.12.6"
  if [ $? -ne 0 ]; then
    echo "rpm not found"
    exit 0
  fi
  echo "docker daemon"
  runtime_root_dir=$(/usr/bin/docker info 2>/dev/null | grep 'Docker Root Dir:' | grep -v "#"  | awk -F: '{print $2}' | sed 's/ //g')

  change_root_dir $runtime_root_dir

  change_log_dir $runtime_root_dir

  change_log_cron_file  $runtime_root_dir

  change_seccomp_profile_root $runtime_root_dir

  change_runtime_cgroupfs $runtime_root_dir

  sed -i "s/--container-runtime=remote/--container-runtime=docker/g" $kubelet_service_path
  sed -i "s/container-runtime-endpoint=unix:\/\/\/var\/run\/pouchcri.sock/container-runtime-endpoint=unix:\/\/\/var\/run\/dockershim.sock/g" $kubelet_service_path

  cgroup_driver=$(docker info 2>/dev/null | grep 'Cgroup Driver:' | grep -v "#"  | awk -F: '{print $2}' | sed 's/ //g')
  sed -i "s/--cgroup-driver=cgroupfs/--cgroup-driver=$cgroup_driver/g" $kubelet_service_path
fi

# modify sigma-slave-start-conf for PAI deploy in et2
modify_start_conf_for_pai_in_et2

systemctl daemon-reload