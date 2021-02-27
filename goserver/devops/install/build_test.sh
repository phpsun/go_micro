#!/bin/bash

EXTRA_PARAMS="-i $WORKSPACE/goserver/devops/install/hosts"

ENV="dev"
USER="root"
if [ "$MODE" = "test" ]; then
    ENV="test"
    USER="sdev"
    EXTRA_PARAMS="$EXTRA_PARAMS -l goservers-test"
elif [ "$MODE" = "dev" ]; then
    EXTRA_PARAMS="$EXTRA_PARAMS -l goservers-dev"
fi

EXTRA_VARS="MODE=$MODE ENV=$ENV USER=$USER WSPACE=$WORKSPACE"

ansible-playbook $WORKSPACE/goserver/devops/playbooks/general-test.yml $EXTRA_PARAMS -e "$EXTRA_VARS"
