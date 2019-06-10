##############################################################
# http://baike.corp.taobao.com/index.php/%E6%B7%98%E5%AE%9Drpm%E6%89%93%E5%8C%85%E8%A7%84%E8%8C%83 #
# http://www.rpm.org/max-rpm/ch-rpm-inside.html              #
##############################################################
Name: t-kubelet-binary
Version:1.1.10
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
alibaba kubelet for k8s


%install
BASE=$OLDPWD/..
cd $BASE

rm -fr $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT/{usr/local/bin,/etc/kubernetes}

install -p -D -m 0755 rpm/kubelet $RPM_BUILD_ROOT/usr/local/bin/kubelet

%clean

%files
%defattr(-,root,root)
/usr/local/bin/kubelet

%pre

%post

%preun

%postun

%changelog
* Mon Jun 10 2019 zibo.hzb
- add spec of kubelet
* Wed Sep 11 2018 fankang.fk
- add userinfo script and pouch-volume
* Wed Jul 25 2018 yaopwei.cyw
- build RPM in packaging host
* Wed Apr 4 2018 yaowei.cyw
- add spec of t-sigma-slave
