#!/bin/bash

os=`awk '{print $7}' /etc/redhat-release 2>/dev/null`
if [ "$os"x != "7.2"x ]; then
  echo "5u6u os skip"
  exit 0
fi

daemon_type="docker"

rpm -qa | grep "pouch"
if [ $? -eq 0 ]; then
	daemon_type="pouch"
else
  rpm -qa | grep "alidocker-1.12.6"
  if [ $? -ne 0 ]; then
    echo "rpm not found"
    exit 0
  fi
fi

mkdir -p /opt/cni/bin/
mkdir -p /etc/cni/net.d/

wget "http://iops.oss-cn-hangzhou-zmf.aliyuncs.com/kuzhi.zm%2Fbin%2Fcni_alinet%2Falinet" -O /tmp/alinet

chmod +x /tmp/alinet

/bin/cp -f /opt/ali-iaas/${daemon_type}/plugins/alinet /opt/ali-iaas/${daemon_type}/plugins/alinet.bak

systemctl stop ${daemon_type}

/bin/cp -f /tmp/alinet /opt/ali-iaas/${daemon_type}/plugins/alinet

systemctl start ${daemon_type}

wget "http://iops.oss-cn-hangzhou-zmf.aliyuncs.com/kuzhi.zm%2Fbin%2Fcni_alinet%2Fcni_alinet" -O /tmp/cni_alinet

/bin/cp -f /tmp/cni_alinet /opt/cni/bin/cni_alinet
chmod +x /opt/cni/bin/cni_alinet

wget "http://iops.oss-cn-hangzhou-zmf.aliyuncs.com/kuzhi.zm%2Fbin%2Fcni_alinet%2Floopback" -O /opt/cni/bin/loopback
chmod +x /opt/cni/bin/loopback

[ ! -f /etc/cni/net.d/net.conf ] && wget "http://iops.oss-cn-hangzhou-zmf.aliyuncs.com/kuzhi.zm%2Fbin%2Fcni_alinet%2Fnet.conf" -O /etc/cni/net.d/net.conf


exit 0
