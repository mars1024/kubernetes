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


# 检查userinfo目录是否存在
if ! [ -d $GRAPH_ROOT/$CTNAME ]; then
    echo "Info: no backup found for $CTNAME"
    exit 0
fi

# 执行拷贝
if [ -d $GRAPH_ROOT/$CTNAME/etc ]; then
    etc_arr=("passwd" "group" "shadow" "sudoers" "ssh/ssh_host_rsa_key" "ssh/ssh_host_dsa_key")
    for one in ${etc_arr[@]}; do
        if [ -s $GRAPH_ROOT/$CTNAME/etc/${one//\//____} ]; then
            [ -f $CTHOME/etc/$one ] && rm -f $CTHOME/etc/$one
            parentDir=$(dirname "$CTHOME/etc/$one")
            [ -d $parentDir ] || mkdir -p $parentDir
            cp -p  $GRAPH_ROOT/$CTNAME/etc/${one//\//____} $CTHOME/etc/$one
            if [[ "$one" == "hostname" ]]; then
                touch $CTHOME/etc/.hosts_place_holder
            fi
        fi
    done
fi

# 拷贝用户目录下的.ssh文件
if [ -d $GRAPH_ROOT/$CTNAME/ssh ]; then
    for one_dir in `ls $GRAPH_ROOT/$CTNAME/ssh/`; do
        one=$GRAPH_ROOT/$CTNAME/ssh/$one_dir/.ssh
        if [ -d $one ]; then
            [ -d $CTHOME/home/${one##*/ssh/} ] && rm -rf $CTHOME/home/${one##*/ssh/}
            parent_dir=`dirname $CTHOME/home/${one##*/ssh/}`
            if [ -d $parent_dir ]; then
                cp -rp $one $CTHOME/home/${one##*/ssh/}
            else
                cp -rp `dirname $one` $CTHOME/home
            fi
        fi
    done
fi

# 处理.hostname.conf
if [ -f $GRAPH_ROOT/$CTNAME/.hostname.conf ]; then
    value=`cat $GRAPH_ROOT/$CTNAME/.hostname.conf`
    if [ -n "$value" ]; then
        # set specified hostname in sysconfig/network, which will be read by rcS when using sbin/init start the vm
        sed -in 's#'HOSTNAME=.*'#'HOSTNAME=${value}'#' $CTHOME/etc/sysconfig/network
        sed -in 's#'HOSTNAME=.*'#'HOSTNAME=${value}'#' $CTHOME/etc/profile.d/dockerenv.sh
        if `hostname | grep -q et2sqa` || `hostname | grep -q et15sqa` || `hostname | grep -q "\.zth"`; then
            echo "$value" > $CTHOME/.hostname.conf
        fi
    fi
fi

# 处理tsar.data
if [ -f $GRAPH_ROOT/$CTNAME/tsar.data ]; then
    [ -d  $CTHOME/var/log/ ] || mkdir -p $CTHOME/var/log/
    cp -af $GRAPH_ROOT/$CTNAME/tsar.data* $CTHOME/var/log/
    rm -rf $GRAPH_ROOT/$CTNAME/tsar.data*
fi

# 处理messages
if [ -f $GRAPH_ROOT/$CTNAME/messages ]; then
    [ -d  $CTHOME/var/log/ ] || mkdir -p $CTHOME/var/log/
    cp -af $GRAPH_ROOT/$CTNAME/messages $CTHOME/var/log/messages
    rm -rf $GRAPH_ROOT/$CTNAME/messages
fi

# 处理route-eth0
if [ -f $GRAPH_ROOT/$CTNAME/route-eth0 ]; then
    cat > $CTHOME/etc/route.tmpl << ABCEOF

[ \$# -eq 2 ] && {
	GW=\$1
	NDEV=\$2
}

cat << EOF
ABCEOF
    sed -e 's/ '$GATEWAY' / $GW /g' -e 's/eth0/$NDEV/g' $GRAPH_ROOT/$CTNAME/route-eth0 >> $CTHOME/etc/route.tmpl
    cat >> $CTHOME/etc/route.tmpl << ABCDEOF
EOF
ABCDEOF
    #rm -f $GRAPH_ROOT/$CTNAME/route-eth0
fi

