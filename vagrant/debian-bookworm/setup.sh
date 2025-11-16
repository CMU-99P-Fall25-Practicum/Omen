#!/bin/bash

# fetch the repo
sudo apt-get update -y
sudo apt-get install -y git python3-full python3-pip \
  libbsd-dev # for openBSD's strlcpy

# install dependencies called on by mn-wifi's install script so it doesn't have to fetch them itself
#sudo apt-get install -y make help2man pyflakes3 python3-pycodestyle tcpdump wpan-tools inetutils-ping
#sudo apt-get install -y python3-six python3-numpy python3-matplotlib python3-bs4 python3-pep8

# Clone repo if it doesn't exist
if [ ! -d "/home/vagrant/mininet-wifi/.git" ]; then
  cd /home/vagrant || exit 1
  rm -rf mininet-wifi
  git clone --depth 1 https://github.com/rflandau/mininet-wifi.git
fi

# consider the repository safe for installation purposes
git config --global --add safe.directory /home/vagrant/mininet-wifi

cd mininet-wifi || exit 1

# create a virtual environment to use
#python3 -m venv mininet_v
#source mininet_v/bin/activate

# add ensure pip and python binaries are in path
PATH=$PATH:/home/vagrant/.local/bin
# disable externally managed environment
pip config set global.break-system-packages true

# mark the repo as safe
git config --global --add safe.directory /home/vagrant/mininet-wifi
# I don't know if this is necessary
sudo git config --global --add safe.directory /home/vagrant/mininet-wifi

# execute the install script, carrying over our venv
# NOTE(rlandau): this is *not* an optimal solution.
# A much better solution would be to make the mn_wifi and mn installer scripts more flexible.
# Specifically, the installer scripts should probably be crafting their own venv or enabling a user to set the python executable to use.
# This doesn't play nicely with sudo as sudo doesn't carry env vars (unless you provide -E).
# Thus, better this be done in the installers than managing the env outside.
#sudo -E env PATH=${PATH} ./util/install-deb12.sh 
#sudo util/install-deb12.sh

# ensure mininet has a clean start up environment
#sudo mn -c