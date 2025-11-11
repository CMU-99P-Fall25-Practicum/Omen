#!/bin/bash

# fetch the repo
sudo apt-get update -y
#sudo apt-get upgrade -y
sudo apt-get install -y git

# install dependencies called on by mn-wifi's install script so it doesn't have to fetch them itself
#sudo apt-get install -y make help2man pyflakes3 python3-pycodestyle tcpdump wpan-tools inetutils-ping
#sudo apt-get install -y python3-six python3-numpy python3-matplotlib python3-bs4 python3-pep8

# Clone repo if it doesn't exist
if [ ! -d "/home/vagrant/mininet-wifi/.git" ]; then
  cd /home/vagrant || exit 1
  rm -rf mininet-wifi
  git clone --depth 1 https://github.com/rflandau/mininet-wifi.git
fi

# add pip-installed items to our path
PATH=$PATH:/home/vagrant/.local/bin

# consider the repository safe for installation purposes
git config --global --add safe.directory /home/vagrant/mininet-wifi


# execute the install script
cd mininet-wifi || exit 1
sudo util/install-deb12.sh

# ensure mininet has a clean start up environment

#sudo mn -c