#!/bin/bash

# fetch the repo
sudo apt-get update -y
#sudo apt-get upgrade -y
sudo apt-get install -y git python3 make build-essential help2man python3-pip

# install dependencies called on by mn-wifi's install script so it doesn't have to fetch them itself
#sudo apt-get install -y make help2man pyflakes3 python3-pycodestyle tcpdump wpan-tools inetutils-ping
#sudo apt-get install -y python3-six python3-numpy python3-matplotlib python3-bs4 python3-pep8

# Clone repo if it doesn't exist
if [ ! -d "/home/vagrant/mininet-wifi/.git" ]; then
  cd /home/vagrant || exit 1
  rm -rf mininet-wifi
  git clone --depth 1 https://github.com/intrig-unicamp/mininet-wifi
fi

# execute the install script
cd mininet-wifi || exit 1
#sudo util/install.sh -Wlnfv

# ensure mininet has a clean start up environment
#sudo mn -c