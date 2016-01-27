# !/bin/bash
#
# lin-build

DEVEL=$HOME/devel
NEXTHOP=$DEVEL/nexthop
export GOPATH=$NEXTHOP

go tool fix $NEXTHOP/src
go tool vet $NEXTHOP/src/sample $NEXTHOP/src/rib-old $NEXTHOP/src/rib $NEXTHOP/src/cli $NEXTHOP/src/rip $NEXTHOP/src/telnet $NEXTHOP/src/command $NEXTHOP/src/fwd
gofmt -s -w $NEXTHOP/src

go install rib-old rib rip

go test command cli

# eof
