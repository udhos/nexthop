#! /bin/bash

if [ "$GOPATH" == "" ]; then
    DEVEL=$HOME/devel
    export GOPATH=$DEVEL/gopath
fi

go_get () {
	local i=$1
	echo go get $i
	go get $i
}

go_get github.com/udhos/nexthop
go_get golang.org/x/net/ipv4
go_get github.com/udhos/netlink
go_get github.com/golang/lint/golint
go_get honnef.co/go/unused/cmd/unused
go_get honnef.co/go/simple/cmd/gosimple
