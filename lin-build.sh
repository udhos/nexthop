# !/bin/bash
#
# lin-build

DEVEL=$HOME/devel
NEXTHOP=$DEVEL/nexthop
export GOPATH=$NEXTHOP

src="addr cli command fwd netorder rib rib-old rip sample sock telnet tools"

#go tool vet $NEXTHOP/src/sample $NEXTHOP/src/rib-old $NEXTHOP/src/rib $NEXTHOP/src/cli $NEXTHOP/src/rip $NEXTHOP/src/telnet $NEXTHOP/src/command $NEXTHOP/src/fwd
for i in $src; do
    j=$NEXTHOP/src/$i
    echo $j
    go tool fix $j
    go tool vet $j
    gofmt -s -w $j
done

$NEXTHOP/bin/unused addr cli command fwd netorder rib rib-old rip sock telnet tools/rip-query

go install rib-old rib rip tools/rip-query

go test command cli addr sock rip netorder

# eof
