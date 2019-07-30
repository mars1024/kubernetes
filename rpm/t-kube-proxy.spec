%global KUBE_MAJOR 1
%global KUBE_MINOR 12
%global KUBE_PATCH 2
%global KUBE_VERSION %{KUBE_MAJOR}.%{KUBE_MINOR}.%{KUBE_PATCH}

%define _prefix /usr/local

# This expands a (major, minor, patch) tuple into a single number so that it
# can be compared against other versions. It has the current implementation
# assumption that none of these numbers will exceed 255.

Name: kube-proxy
Version: %{KUBE_VERSION}
Release: %(echo $RELEASE)
Summary: AntCloud Kubernetes Proxy
License: Commercial

URL: https://kubernetes.io

BuildRequires: systemd
BuildRequires: curl
Requires: iptables >= 1.4.21
Requires: socat
Requires: util-linux
Requires: ethtool
Requires: iproute
Requires: ebtables
Requires: conntrack

%description
The Kubernetes Proxy

%prep
# Assumes the builder has overridden sourcedir to point to directory
# with this spec file. (where these files are stored) Copy them into
# the builddir so they can be installed.
# This is a useful hack for faster Docker builds when working on the spec or
# with locally obtained sources.
#
# Example:
#   spectool -gf kube-proxy.spec
#   rpmbuild --define "_sourcedir $PWD" -bb kube-proxy.spec
#

%install
BASE=$OLDPWD/..
cd $BASE

rm -fr $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT/{usr/local/bin,/etc/kubernetes,/etc/kubernetes/config}

install -p -D -m 0755 rpm/kube-proxy $RPM_BUILD_ROOT/usr/local/bin/kube-proxy
install -p -D -m 0755 rpm/systemd/kube-proxy.service  $RPM_BUILD_ROOT/etc/systemd/system/
install -p -D -m 0755 rpm/systemd/kube-proxy.service.d/kube-proxy.conf  $RPM_BUILD_ROOT/etc/systemd/system/kube-proxy.service.d/

%files
%defattr(-,root,root)
/usr/local/bin/kube-proxy
/etc/systemd/system/kube-proxy.service
/etc/systemd/system/kube-proxy.service.d

%changelog
* Wed Jun 11 2019 Di Xu <stephen.xd@antfin.com> - 1.12.2
- Init rpm for kube-proxy
