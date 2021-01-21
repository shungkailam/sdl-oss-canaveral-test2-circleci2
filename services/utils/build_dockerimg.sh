#!/bin/bash -x

export EDGEPROFILEDIR="$HOME/.edgerunner"
export DOCKERTAGSFILE="$EDGEPROFILEDIR/dockerImageTags"
export DOCKERTAGSLOCK="$EDGEPROFILEDIR/dockerImageTags.lock"

# Find the OS
unameOut="$(uname -s)"
case "${unameOut}" in
    Linux*)     MACHINE=linux;;
    Darwin*)    MACHINE=darwin;;
    *)          MACHINE="UNKNOWN:${unameOut}"
esac

# Find the distro of linux
case "$MACHINE" in
    linux*)
        if [ -f /etc/os-release ]; then
            . /etc/os-release
            LINUX_DISTRO=$NAME
        else
            LINUX_DISTRO="Unsupported"
        fi

        if [ $LINUX_DISTRO != "Ubuntu" ]; then
            echo "This distro of linux is not supported yet"
            exit 1
        fi;;
    *)
        echo "Only Ubuntu is supported for now"
        exit 1
esac

function install_docker {
    if type -P docker > /dev/null; then
        echo "docker is already installed"
        return
    fi
    echo 'Installing docker'

    sudo apt-get update
    sudo apt-get install -y apt-transport-https ca-certificates \
         curl software-properties-common
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
    sudo add-apt-repository \
        "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
        $(lsb_release -cs) \
        stable"

    sudo apt-get update
    sudo apt-get install -y docker-ce    
    sudo usermod -a -G docker $USER
    newgrp docker
}

function create_tag {
    key=$1
    if [ -f $DOCKERTAGSFILE ]; then
        # Check if key exists, if not add it 
        if grep -q $key $DOCKERTAGSFILE
        then 
            echo "file exists" > /dev/null
        else
            echo "$key=0" >> $DOCKERTAGSFILE
        fi
        return
    fi
    mkdir -p $EDGEPROFILEDIR
    echo "$key=0" >> $DOCKERTAGSFILE
}

function get_nexttag {
    key=$1

    # Create initial tag if required
    create_tag $key

    # Update tag
    curtag="$(awk -F'[=]' "/^$key/ {print \$2}" $DOCKERTAGSFILE | awk '{print $1}')"
    curtag="$((curtag + 1))"
    sed -i -r "s/($key *= *).*/\1$curtag/" $DOCKERTAGSFILE
    echo $curtag
}

function lock_tagfile {
    # create dir as required
    mkdir -p $EDGEPROFILEDIR

    if ln -s $EDGEPROFILEDIR $DOCKERTAGSLOCK 2> /dev/null; then
        return
    else
        echo "lock [$DOCKERTAGSLOCK] on tag file exists, retry in few seconds"
        exit 1
    fi
}

function unlock_tagfile {
    mv $DOCKERTAGSLOCK ${DOCKERTAGSLOCK}_tmp 
    rm ${DOCKERTAGSLOCK}_tmp 
}