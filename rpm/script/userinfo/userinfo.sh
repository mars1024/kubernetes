#! /bin/bash

# backup <container-id> <unique-container-name>
# restore <container-id> <unique-container-name>
# check <unique-container-name>
# delete <unique-container-name>

CURRENT_PATH=$(cd `dirname $0`;pwd)

# 获取graph主目录
GRAPH_ROOT="/home/userinfo"

case $1 in
backup)
    CONTAINER_ID=$2

    DIR_NAME=$3

    $CURRENT_PATH/backup_user.sh $CONTAINER_ID $DIR_NAME
    ;;
restore)
    CONTAINER_ID=$2

    DIR_NAME=$3

    $CURRENT_PATH/restore_user.sh $CONTAINER_ID $DIR_NAME
    ;;
check)
    DIR_NAME=$2
    if [ ! -d "$GRAPH_ROOT/$DIR_NAME" ];then
        exit 1
    fi
    ;;
delete)
    DIR_NAME=$2
    if [ -d "$GRAPH_ROOT/$DIR_NAME" ];then
        rm -rf "$GRAPH_ROOT/$DIR_NAME"
    fi
    ;;
esac
