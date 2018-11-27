#!/bin/bash
# Author : lvshun.ls

if [[ $# -lt 3 ]]; then
	echo "need target_dir log_level remain_count"
	exit -1
fi

target_dir=$1
log_level=$2
remain_count=$3

if ((${remain_count} <= 0)); then
	echo "remain_count need greater than 0!"
	exit -2
fi

# judge if target_dir exists
if ! [[ -d ${target_dir} ]]; then
	echo "dir $target_dir not exists!"
	exit -3
fi

# filter files under target_dir
to_sort_files=()
todo_count=0

exclude_files=()
exclude_count=0

target_files=$(ls ${target_dir}|awk '{print i$0}' i=${target_dir}'/')

condition="^${target_dir}/sigma-slave\.[^\.]+\.[^\.]+\.log\.${log_level}\.[^\.]+\.[^\.]+$"
for f in ${target_files[@]}
do
	if ! [[ -f ${f} ]]; then
		continue
	elif [[ -L ${f} ]]; then
		link_file=`readlink ${f}`
		if [[ "${link_file}" =~ ^[^/]+ ]]; then
			link_file=${target_dir}'/'${link_file}
		fi
		if [[ "${link_file}" =~ ${condition} ]]; then
			exclude_files[exclude_count]=${link_file}
			((exclude_count+=1))
		fi
		continue
	elif ! [[ "${f}" =~ ${condition} ]]; then
		continue
	fi

	to_sort_files[todo_count]=${f}
	((todo_count+=1))
done

# sort
sorted_files=$(for f in ${to_sort_files[@]}; do echo $f; done | sort -t "." -k 6 -r)

to_del_files=()
todel_count=0
filtered_count=0

for f in ${sorted_files[@]}
do
	if [[ "${exclude_files[@]}" = *"${f}"* ]]; then
		continue
	elif [[ ${filtered_count} -lt ${remain_count} ]]; then
		((filtered_count+=1))
		continue
	fi

	to_del_files[todel_count]=${f}
	((todel_count+=1))
done

# print and clean
for f in ${to_del_files[@]}
do
	ionice -c2 -n7 rm -f "${f}"
done

