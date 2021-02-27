#!/bin/bash

BUILD_HOST="127.0.0.1"
EXTRA_PARAMS="-i $WORKSPACE/goserver/devops/install/hosts"

ENV="prod"
HOST_GROUP=$ENV
if [ "$PROGRAM" = "wallet" ]; then
    HOST_GROUP=$PROGRAM
elif [ "$PROGRAM" = "internal" ]; then
    HOST_GROUP=$PROGRAM
elif [ "$MODE" = "grayscale" ]; then
    HOST_GROUP="gray"
fi

EXTRA_VARS="MODE=$MODE PROGRAM=$PROGRAM ENV=$ENV BHOST=$BUILD_HOST WSPACE=$WORKSPACE HGROUP=$HOST_GROUP"

if [ "$WHICH_COMMIT" != "" ]; then
    git checkout $WHICH_COMMIT
fi

if [ "$ACTION" = "restore" ]; then
    ansible-playbook $WORKSPACE/goserver/devops/playbooks/general-restore.yml $EXTRA_PARAMS -e "$EXTRA_VARS"
else
    rsync -av --delete --exclude=.git $WORKSPACE/ $BUILD_HOST:/data/build/goserver_src/

    ssh $BUILD_HOST "
	cd /data/build/goserver_src/goserver/src/${PROGRAM}_server
	go build -o ${PROGRAM}_server main.go || exit 1
    "

    ansible-playbook $WORKSPACE/goserver/devops/playbooks/general.yml $EXTRA_PARAMS -e "$EXTRA_VARS"
fi
