#!/bin/bash
#****************************************************************#
# ScriptName: clean.sh
# Author: yaowei.cyw@alibaba-inc.com
# Create Date: 2018-08-24 13:30
# Modify Author: $SHTERM_REAL_USER@alibaba-inc.com
# Modify Date: 2018-08-24 13:30
# Function: clean sigma-slave unused certificate
#***************************************************************#

if [[ $# != 1 ]]; then
  echo "should include sigma-slave certificate dir."
  exit 1
fi


site=$(/bin/bash -c hostinfo --cmdb 2>/dev/null | grep logicRegion | awk '{print $2}' |  sed 's/ //g' | tr 'A-Z' 'a-z')
if [[ -z "$site" ]]; then
	echo "can't get site, exit"
	exit 1
fi

sigma_slave_certificate_dir=$1

cd  $sigma_slave_certificate_dir

target_files=$(ls ${sigma_slave_certificate_dir}|grep sigma-slave-certificate.conf)
condition="^${site}-sigma-slave-certificate.conf"

for f in ${target_files[@]}
do
	if [[ "${f}" =~ ${condition} ]]; then
		mv ${site}-sigma-slave-certificate.conf sigma-slave-certificate.conf
		continue
	fi
	rm -r ${f}
done
exit 0