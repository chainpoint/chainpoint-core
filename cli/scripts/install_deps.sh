#!/bin/bash

install_deps(){
  echo "Running installation as"
  echo "Username: $USER"
  echo "    EUID: $EUID"
  if [[ "$OSTYPE" == "linux-gnu" ]]; then

    sudo bash -c "apt-get update -y && apt-get install -y wget build-essential libsnappy-dev libcap2-bin "

    wget https://github.com/google/leveldb/archive/v1.20.tar.gz && \
    tar -zxvf v1.20.tar.gz && \
    cd leveldb-1.20/ && \
    make && \
    cp -r out-static/lib* out-shared/lib* /usr/local/lib/ && \
    cd include/ && \
    cp -r leveldb /usr/local/include/ && \
    ldconfig && \
    rm -f v1.20.tar.gz

  elif [[ "$OSTYPE" == "darwin"* ]]; then
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    brew install snappy wget
    cd /usr/local/lib/
    ln -s /usr/local/Cellar/snappy/1.1.1/lib/libsnappy.1.dylib
    ln -s /usr/local/Cellar/snappy/1.1.1/lib/libsnappy.a
    ln -s /usr/local/Cellar/snappy/1.1.1/lib/libsnappy.dylib
    update_dyld_shared_cache

    wget https://github.com/google/leveldb/archive/v1.20.tar.gz && \
    tar -zxvf v1.20.tar.gz && \
    cd leveldb-1.20/ && \
    make && \
    cp -r out-static/lib* out-shared/lib* /usr/local/lib/ && \
    cd include/ && \
    cp -r leveldb /usr/local/include/ && \
    update_dyld_shared_cache && \
    rm -f v1.20.tar.gz
  fi
}

sudo bash -c "$(declare -f install_deps); install_deps"
