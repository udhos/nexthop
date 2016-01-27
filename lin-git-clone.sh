#! /bin/bash

export DEVEL=$HOME/devel
export NEXTHOP=$DEVEL/nexthop
export GOPATH=$NEXTHOP

[ -d $DEVEL ] || mkdir -p $DEVEL
cd $DEVEL

git clone https://github.com/udhos/nexthop

go_get () {
	local i=$1
	echo go get $i
	go get $i
}

#go_get code.google.com/p/go.net/ipv4
#go_get golang.org/x/net/ipv4
go_get github.com/udhos/netlink
