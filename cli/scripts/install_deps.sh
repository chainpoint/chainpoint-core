#!/bin/bash

if [ -x "$(command -v docker)" ]; then
    echo "Docker already installed"
else
    echo "Install docker"
    curl -fsSL https://get.docker.com -o get-docker.sh
    bash get-docker.sh
    sudo usermod -aG docker $USER
fi

if [[ "$OSTYPE" == "linux-gnu" ]]; then
    sudo apt-get -qq update -y
    sudo apt-get -qq install -y apt-utils curl dirmngr apt-transport-https lsb-release ca-certificates g++
    curl -sL https://deb.nodesource.com/setup_12.x | sudo -E bash -
    sudo apt-get -qq install -y docker-compose git make jq nodejs openssl
    sudo apt-get -qq install -y npm || echo "NPM already installed with nodejs"
    curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -
    if ! [ -x "$(command -v yarn)" ]; then
        echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
        sudo apt-get update -y && sudo apt-get install -y --no-install-recommends yarn
    fi
    sudo curl -L "https://github.com/docker/compose/releases/download/1.25.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    sudo ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose || echo Binary is not at usual location or is already linked
elif [[ "$OSTYPE" == "darwin"* ]]; then
    ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"
    brew install caskroom/cask/brew-cask
    brew cask install docker-toolbox
    brew install jq
    brew install homebrew/core/make
    brew install git
    brew install node
    brew install yarn
    brew install openssl
fi

yarn