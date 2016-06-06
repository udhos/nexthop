# !/bin/bash
#
# lin-build.sh

if [ "$GOPATH" == "" ]; then
    DEVEL=$HOME/devel
    export GOPATH=$DEVEL/gopath
fi

PATH=$GOPATH/bin:$PATH
NHPATH=github.com/udhos/nexthop
NEXTHOP=$GOPATH/src/$NHPATH

src="addr bgp cli command fwd netorder rib rib-old rip sock telnet tools           sample"
unu="addr bgp cli command fwd netorder rib rib-old rip sock telnet tools/rip-query"

msg() {
    echo $*
}

fix() {
    local i=$1
    j=$NEXTHOP/$i
    msg build: go tool fix $j
    go tool fix $j
    msg build: go tool vet $j
    go tool vet $j
    msg build: gofmt -s -w $j
    gofmt -s -w $j
    k=$NHPATH/$i
    msg build: gosimple $k
    gosimple $k
    #golint $j     ;# golint is verbose, enable only when actually needed
}

run_unused() {
    for i in $unu; do
	j=$NHPATH/$i
	msg build: unused $j
	unused $j ;# unused is slow
    done
}

static() {
    for i in $src; do
	fix $i	
    done

    run_unused ;# unused is slow
}

install() {

    inst='rib-old rib rip bgp tools/rip-query'

    for i in $inst; do
	j=$NHPATH/$i
	msg install: go install $j
	go install $j
    done
}

test() {
    save=$PWD
    cd $NEXTHOP
    test_dirs=`ls */*_test.go | awk -F/ '{ print $1 }'`
    cd $save

    for i in $test_dirs; do
	j=$NHPATH/$i
	#msg test: go test $j
	go test $j
    done
}

static
install
test

# eof
