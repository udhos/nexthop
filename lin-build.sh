# !/bin/bash
#
# lin-build

DEVEL=$HOME/devel
NEXTHOP=$DEVEL/nexthop
export GOPATH=$NEXTHOP

PATH=$NEXTHOP/bin:$PATH

src="addr bgp cli command fwd netorder rib rib-old rip sample sock telnet tools"

for i in $src; do
    j=$NEXTHOP/src/$i
    echo $j
    go tool fix $j
    go tool vet $j
    gofmt -s -w $j
    gosimple $i
    #golint $j     ;# golint is verbose, enable only when actually needed
done

#unused addr cli command fwd netorder rib rib-old rip sock telnet tools/rip-query

go install rib-old rib rip bgp tools/rip-query

go test bgp command cli addr sock rip netorder

# eof
