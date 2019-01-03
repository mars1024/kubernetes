##############################################################
# http://baike.corp.taobao.com/index.php/%E6%B7%98%E5%AE%9Drpm%E6%89%93%E5%8C%85%E8%A7%84%E8%8C%83 #
# http://www.rpm.org/max-rpm/ch-rpm-inside.html              #
##############################################################
Name: t-sigma-slave
Version:1.0.21
Release: %(echo $RELEASE)
# if you want use the parameter of rpm_create on build time,
# uncomment below
Summary: alibaba kubelet.
Group: alibaba/application
License: Commercial
AutoReqProv: none
%define _prefix /usr/local
%define _systemd /etc/systemd/system
%define rhel %(/usr/lib/rpm/redhat/dist.sh --distnum)


BuildArch:x86_64
BuildRequires: t-db-golang = 1.8.4-20180731153834


%description
# if you want publish current svn URL or Revision use these macros
alibaba kubelet for k8s and sigma

%prep
%if %{rhel} < 7
    echo "sigma-slave only build for 7u alios."
    exit 1
%endif


%build


%install
BASE=$OLDPWD/..
cd $BASE

rm -fr $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT/{usr/local/bin,/etc/systemd/system,/etc/systemd/system/sigma-slave.service.d,/etc/kubernetes,/etc/kubernetes/check}

install -p -D -m 0755 rpm/sigma-slave $RPM_BUILD_ROOT/usr/local/bin/sigma-slave
install -p -D -m 0755 rpm/systemd/sigma-slave.service  $RPM_BUILD_ROOT/etc/systemd/system/
install -p -D -m 0755 rpm/systemd/sigma-slave.service.d/sigma-slave-start.conf  $RPM_BUILD_ROOT/etc/systemd/system/sigma-slave.service.d/
install -p -D -m 0755 rpm/script/release/*  $RPM_BUILD_ROOT/etc/kubernetes/
install -p -D -m 0755 rpm/script/userinfo/*  $RPM_BUILD_ROOT/etc/kubernetes/
install -p -D -m 0755 rpm/script/checklist/*.sh  $RPM_BUILD_ROOT/etc/kubernetes/
install -p -D -m 0755 rpm/script/checklist/check/*  $RPM_BUILD_ROOT/etc/kubernetes/check/
install -p -D -m 0755 rpm/certificate/*  $RPM_BUILD_ROOT/etc/kubernetes/
install -p -D -m 0644 rpm/conf/sigma-slave-clean-log.cron $RPM_BUILD_ROOT/etc/cron.d/sigma-slave-clean-log


%clean

%files
%defattr(-,root,root)
/usr/local/bin/sigma-slave
/etc/systemd/system/sigma-slave.service
/etc/systemd/system/sigma-slave.service.d
/etc/cron.d/sigma-slave-clean-log
/etc/kubernetes/*
/etc/kubernetes/check/*


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

    sh -x /etc/kubernetes/check/host-slave-keep.sh start >> /tmp/sigma-slave-error.log 2>&1
    if [[ $? -ne 0 ]]; then
        exit 1
    fi

    sh -x /etc/kubernetes/clean-certificate.sh /etc/kubernetes >> /tmp/sigma-slave-error.log 2>&1
    sh -x /etc/kubernetes/modify_start_up_params.sh /etc/systemd/system/sigma-slave.service.d/sigma-slave-start.conf /etc/cron.d/sigma-slave-clean-log >> /tmp/sigma-slave-error.log  2>&1

    systemctl daemon-reload
    systemctl enable sigma-slave
    systemctl restart sigma-slave

    sleep 3
    sh -x /etc/kubernetes/check.sh start >> /tmp/sigma-slave-error.log 2>&1

%preun

%postun

%changelog
* Thu Dec 27 2018 yaowei.cyw
- add check list
* Wed Jul 25 2018 yaowei.cyw
- build RPM in packaging host
* Wed Apr 4 2018 yaowei.cyw
- add spec of t-sigma-slave
