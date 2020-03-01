#!/bin/sh

if [ -n "${DEBUG_SHELL+set}" ]; then
  set -x
fi

export SSH_PARAMS='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/tmp/known_hosts'

mkdir /tmp/ssh 2>&1

echo ${SSH_PK} | base64 -d > /tmp/ssh/deploy_key
eval $(ssh-agent -s)
chmod 600 /tmp/ssh/deploy_key
ssh-add /tmp/ssh/deploy_key

ssh ${SSH_PARAMS} ${SSH_USER}@${SSH_HOST} sudo systemctl stop fcc
scp ${SSH_PARAMS} bin/fcc_linux ${SSH_USER}@${SSH_HOST}:/opt/fcc/fcc
ssh ${SSH_PARAMS} ${SSH_USER}@${SSH_HOST} sudo systemctl start fcc

rm -rf /tmp/ssh
kill ${SSH_AGENT_PID}
