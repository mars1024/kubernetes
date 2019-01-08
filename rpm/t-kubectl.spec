##############################################################
# http://baike.corp.taobao.com/index.php/%E6%B7%98%E5%AE%9Drpm%E6%89%93%E5%8C%85%E8%A7%84%E8%8C%83 #
# http://www.rpm.org/max-rpm/ch-rpm-inside.html              #
##############################################################
Name: t-sigma-slave
Version:1.0.21
Release: %(echo $RELEASE)
# if you want use the parameter of rpm_create on build time,
# uncomment below
Summary: alibaba kubectl.
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
alibaba kubectl for k8s and sigma

%prep
%if %{rhel} < 7
    echo "kubectl only build for 7u alios."
    exit 1
%endif


%build


%install
BASE=$OLDPWD/..
cd $BASE

rm -fr $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT/{usr/local/bin}

install -p -D -m 0755 rpm/kubectl $RPM_BUILD_ROOT/usr/local/bin/kubectl



%clean

%files
%defattr(-,root,root)
/usr/local/bin/kubectl


%pre

%post

%preun

%postun

%changelog
* Wed Jan 8 2019 zhongyuan.zxy
- add spec of t-sigma-slave
