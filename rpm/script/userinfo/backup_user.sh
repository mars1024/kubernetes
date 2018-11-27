#! /bin/bash

if [ $# -lt 2 ]; then
    echo "Usage: $0 ContainerID_or_Name Dir_Name"
    exit 1
fi

CT_ID=$1
CTNAME=$2

# 检查runtime，默认为Docker
RUNTIME="Docker"
if command "pouch" &> /dev/null; then
    RUNTIME="Pouch"
fi

# 检测容器是否存在
# grep -q是安静模式
if [ "$RUNTIME" = "Docker" ]; then
    if ! `docker ps -a --no-trunc | grep -q $CT_ID`; then
        echo "Fail: no container found by symbol $CT_ID"
        exit 2
    fi
else
    if ! `pouch ps -a --no-trunc | grep -q $CT_ID`; then
        echo "Fail: no container found by symbol $CT_ID"
        exit 2
    fi
fi

# 获取/var/lib/docker/overlay/2276e1ff8b770f9cca08fd3f6f99d4587ddd1325a6a13b4463c94b068a6e9e0d/upper
if [ "$RUNTIME" = "Docker" ]; then
    CTHOME=`docker inspect -f "{{.GraphDriver.Data.UpperDir}}" $CT_ID` 
else
    CTHOME=`pouch inspect -f "{{.GraphDriver.Data.UpperDir}}" $CT_ID`
fi

# 指定备份目录
GRAPH_ROOT="/home/userinfo"
if [ ! -d $GRAPH_ROOT ];then
    mkdir -p $GRAPH_ROOT
fi

# 指定临时备份目录
GRAPH_ROOT_TMP="/tmp/userinfo"
if [ ! -d $GRAPH_ROOT_TMP ];then
    mkdir -p $GRAPH_ROOT_TMP
fi

# 在graph主目录下创建$CTNAME/etc
[ -d $GRAPH_ROOT_TMP/$CTNAME/etc ] || mkdir -p $GRAPH_ROOT_TMP/$CTNAME/etc

# 需要备份的文件
etc_arr=("passwd" "group" "shadow" "sudoers" "ssh/ssh_host_rsa_key" "ssh/ssh_host_dsa_key")

# 备份
for one_file in ${etc_arr[@]}; do
    [ -f $GRAPH_ROOT_TMP/$CTNAME/etc/${one_file//\//____} ] && rm -f $GRAPH_ROOT_TMP/$CTNAME/etc/${one_file//\//____}
    [ -f $CTHOME/etc/$one_file ] && cp -p $CTHOME/etc/$one_file $GRAPH_ROOT_TMP/$CTNAME/etc/${one_file//\//____}
done

# 处理容器/home目录下的内容
if [ -d $CTHOME/home ]; then
    # 轮询每一个/home下的用户目录
    for one in `ls $CTHOME/home`; do
        # 如果存在.ssh目录
        if [ -d $CTHOME/home/$one/.ssh ]; then
            one_file=$CTHOME/home/$one/.ssh
            [ -d $GRAPH_ROOT_TMP/$CTNAME/ssh/${one_file##*/home/} ] && rm -rf $GRAPH_ROOT_TMP/$CTNAME/ssh/${one_file##*/home/}
            parent_dir=`dirname $GRAPH_ROOT_TMP/$CTNAME/ssh/${one_file##*/home/}`
            if ! [ -d $parent_dir ]; then
                read usr grp< <(ls -ld $one_file/.. | awk '{print $3,$4}')
                mkdir -p $parent_dir
                chown -R $usr:$grp $parent_dir
            fi
            # 拷贝
            cp -rp $one_file $GRAPH_ROOT_TMP/$CTNAME/ssh/${one_file##*/home/}
        fi
    done
fi

# 处理/etc/sysconfig/network
if [ -f $CTHOME/etc/sysconfig/network ]; then
    hn=`awk -F "[ \t=#]+" '$1=="HOSTNAME"{print $2}' $CTHOME/etc/sysconfig/network`
    if [ -n "$hn" ]; then
        echo "$hn" > $GRAPH_ROOT_TMP/$CTNAME/.hostname.conf
    fi
fi

# 处理/etc/rc3.d/S80umount
if [ -f $CTHOME/etc/rc3.d/S80umount ]; then
    [ -d $CTHOME/home/admin/cai/alivmcommon ] && cp -rp $CTHOME/home/admin/cai/alivmcommon/* $GRAPH_ROOT_TMP/$CTNAME/vmcommon/
    [ -d $CTHOME/home/admin/cai/top_foot_vm ] && cp -rp $CTHOME/home/admin/cai/top_foot_vm/* $GRAPH_ROOT_TMP/$CTNAME/top_foot_vm/
fi

# 处理route-eth0文件
if [ -f $CTHOME/etc/sysconfig/network-scripts/route-eth0 ] && `fgrep -q "default" $CTHOME/etc/sysconfig/network-scripts/route-eth0`; then
    cp -af $CTHOME/etc/sysconfig/network-scripts/route-eth0 $GRAPH_ROOT_TMP/$CTNAME/route-eth0
fi

# 处理/var/log/tsar.data
if [ -f $CTHOME/var/log/tsar.data ]; then
    cp -af $CTHOME/var/log/tsar.data* $GRAPH_ROOT_TMP/$CTNAME/
fi

# 处理/var/log/messages
if [ -f $CTHOME/var/log/messages ]; then
    cp -af $CTHOME/var/log/messages $GRAPH_ROOT_TMP/$CTNAME/messages
fi

# 把tmp目录移到GRAPH_ROOT中
rm -rf $GRAPH_ROOT/$CTNAME
mv $GRAPH_ROOT_TMP/$CTNAME $GRAPH_ROOT/$CTNAME

