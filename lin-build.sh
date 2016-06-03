# !/bin/bash
#
# lin-build

DEVEL=$HOME/devel
export GOPATH=$DEVEL/gopath
PATH=$GOPATH/bin:$PATH
NHPATH=github.com/udhos/nexthop
NEXTHOP=$GOPATH/src/$NHPATH

src="addr bgp cli command fwd netorder rib rib-old rip sock telnet tools           sample"
unu="addr bgp cli command fwd netorder rib rib-old rip sock telnet tools/rip-query"

fix() {
    local i=$1
    j=$NEXTHOP/$i
    echo build: go tool fix $j
    go tool fix $j
    echo build: go tool vet $j
    go tool vet $j
    echo build: gofmt -s -w $j
    gofmt -s -w $j
    k=$NHPATH/$i
    echo build: gosimple $k
    gosimple $k
    #golint $j     ;# golint is verbose, enable only when actually needed
}
static() {
for i in $src; do
    fix $i	
done

for i in $unu; do
    j=$NHPATH/$i
    echo build: unused $j
    unused $j
done
}

install() {

inst='rib-old rib rip bgp tools/rip-query'

for i in $inst; do
    j=$NHPATH/$i
    echo install: go install $j
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
    echo test: go test $j
    go test $j
done
}

static
install
test

# eof
