##############################################################
# http://baike.corp.taobao.com/index.php/%E6%B7%98%E5%AE%9Drpm%E6%89%93%E5%8C%85%E8%A7%84%E8%8C%83 #
# http://www.rpm.org/max-rpm/ch-rpm-inside.html              #
##############################################################
Name: t-sigma-slave-binary
Version:1.1.0
Release: %(echo $RELEASE)
# if you want use the parameter of rpm_create on build time,
# uncomment below
Summary: alibaba kubelet.
Group: alibaba/application
License: Commercial
AutoReqProv: none
%define _prefix /usr/local

BuildArch:x86_64
BuildRequires: t-db-golang = 1.8.4-20180731153834


%description
# if you want publish current svn URL or Revision use these macros
alibaba kubelet for k8s and sigma


%install
BASE=$OLDPWD/..
cd $BASE

rm -fr $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT/{usr/local/bin,/etc/kubernetes,/etc/kubernetes/check}
mkdir -p $RPM_BUILD_ROOT/usr/libexec/kubernetes/kubelet-plugins/volume/exec/alipay~pouch-volume

install -p -D -m 0755 rpm/sigma-slave $RPM_BUILD_ROOT/usr/local/bin/sigma-slave
install -p -D -m 0755 rpm/script/userinfo/*  $RPM_BUILD_ROOT/etc/kubernetes/
install -p -D -m 0755 rpm/script/checklist/*.sh  $RPM_BUILD_ROOT/etc/kubernetes/
install -p -D -m 0755 rpm/script/checklist/check/*  $RPM_BUILD_ROOT/etc/kubernetes/check/
install -p -D -m 0755 rpm/script/volume/pouch-volume  $RPM_BUILD_ROOT/usr/libexec/kubernetes/kubelet-plugins/volume/exec/alipay~pouch-volume/
install -p -D -m 0755 rpm/script/release/sigma-slave-clean-expire-logs.sh $RPM_BUILD_ROOT/etc/kubernetes/

%clean

%files
%defattr(-,root,root)
/usr/local/bin/sigma-slave
/etc/kubernetes/*
/etc/kubernetes/check/*
/usr/libexec/kubernetes/kubelet-plugins/volume/exec/alipay~pouch-volume/pouch-volume

%pre

%post
    cd /etc/kubernetes
    sh -x /etc/kubernetes/check/check-host-env.sh > /tmp/sigma-slave-error.log 2>&1
    if [[ $? -ne 0 ]]; then
        exit 1
    fi

    sh -x /etc/kubernetes/pre-rpm-upgrade.sh >> /tmp/sigma-slave-error.log 2>&1
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
%preun

%postun

%changelog
* Wed Sep 11 2018 fankang.fk
- add userinfo script and pouch-volume
* Wed Jul 25 2018 yaopwei.cyw
- build RPM in packaging host
* Wed Apr 4 2018 yaowei.cyw
- add spec of t-sigma-slave
