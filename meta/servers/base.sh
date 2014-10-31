#!/bin/bash

# base install script for a basic docker host

echo 'upgrading the current host...'
apt-get update
apt-get upgrade -y

echo 'install base dependencies...'
apt-get install -y cgroup-lite aufs-tools git htop supervisor curl

echo 'install docker nightly version...'
curl -o /usr/local/bin/docker https://test.docker.com/builds/Linux/docker-latest
chmod +x /usr/local/bin/docker

echo 'setup supervisor...'
echo '[program:docker]
command=/usr/local/bin/docker -dD -s aufs
autostart=true
autorestart=true
stdout_logfile=/var/log/docker.log
redirect_stderr=true
numprocs=1' > /etc/supervisor/conf.d/docker.conf

supervisorctl reload
