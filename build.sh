#!/bin/bash

#
# Utility
#
install_package()
{
    local PKG_NAME=$1
    local PPA_NAME=$2

    if [ "$PKG_NAME" == "" ]; then
        echo "ERROR: no package name specified"
        return
    fi

    dpkg -V $PKG_NAME 2>/dev/null
    if [ "$?" == "0" ]; then
        return
    fi

    if [ "$PPA_NAME" != "" ]; then
        sudo add-apt-repository -y -r $PPA_NAME
        sudo add-apt-repository -y $PPA_NAME
        sudo apt-get update -q
    fi

    sudo apt-get install -y $PKG_NAME
}

#
# Development Environment: Directory Layout
#
# $HOME
#  dev/         - DEV_ROOT - GOPATH
#    src/       - SRC_ROOT
#      DNA_POW/ - SRC_PATH
#        deps/  - Depended C source code
#
setenv()
{
    install_package git
    install_package software-properties-common
    install_package golang-1.8-go ppa:gophers/archive
    install_package glide ppa:masterminds/glide

    export SCRIPT_PATH=$(cd $(dirname $BASH_SOURCE); pwd)
    export SRC_PATH=$SCRIPT_PATH

    export DEV_ROOT=$(cd $SCRIPT_PATH/../..; pwd)
    export SRC_ROOT=$DEV_ROOT/src

    export GOROOT=/usr/lib/go-1.8
    export GOPATH=$DEV_ROOT

    export PATH=$GOROOT/bin:$PATH
    export PATH=$GOBIN:$PATH

    NCPU=1
    if [ "$(uname -s)" == "Linux" ]; then
        NCPU=$(($(grep '^processor' /proc/cpuinfo | wc -l) * 2))
    elif [ "$(uname -s)" == "Darwin" ]; then
        NCPU=$(($(sysctl -n hw.ncpu) * 2))
    fi
}

build_dependencies()
{
    mkdir -p $SRC_PATH/deps/
    cd $SRC_PATH/deps/

    echo "Downloading zeromq..."
    wget -q "https://github.com/zeromq/libzmq/releases/download/v4.2.2/zeromq-4.2.2.tar.gz" -O zeromq-4.2.2.tar.gz
    tar xf zeromq-4.2.2.tar.gz

    echo "Building zeromq..."
    cd zeromq-4.2.2
    ./configure -q
    make -s -j$NCPU

    echo "Installing zeromq..."
    sudo make -s install
    sudo ldconfig
}

build()
{
    cd $SRC_PATH
    glide install
    make
}

usage()
{
    echo "Usage: $(basename $0)"
    echo "Build this project"
}

#
# Main
#

#
# Check OS version
#
OS_NAME=$(uname -s)
if [ "$OS_NAME" == "Linux" ]; then
    OS_DETAIL="$(lsb_release -i -s) $(lsb_release -r -s) $(uname -m)"
elif [ "$OS_NAME" == "Darwin" ]; then
    OS_DETAIL="$(uname -srm)"
fi

if [ "$OS_DETAIL" == "Ubuntu 16.04 x86_64" ]; then
    echo "$OS_DETAIL: Supported"
else
    echo "ERROR: $OS_DETAIL have not been tested"
    exit
fi

if [ "$1" == "-h" ]; then
    usage
    exit
fi

setenv
build_dependencies
build
