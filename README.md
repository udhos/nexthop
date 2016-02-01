nexthop
=======

Dynamic Internet Routing Suite in Go

quick start - Linux
===================

1. install Go from https://golang.org/dl/

2. set GOPATH: set GOPATH=$HOME/devel/nexthop

3. enter development dir: cd $HOME/devel

4. get nexthop: git clone https://github.com/udhos/nexthop

5. build: $HOME/devel/nexthop/lin-build.sh

6. run daemons

rib: sudo $HOME/devel/nexthop/bin/rib
rip: sudo $HOME/devel/nexthop/bin/rip

7. access the daemon CLI with TELNET

rib: telnet localhost 2001
rip: telnet localhost 2002

quick start - Windows 8
=======================

1. install Go from https://golang.org/dl/

2. set GOPATH: set GOPATH=c:\tmp\devel\nexthop

3. enter development dir: cd c:\tmp\devel

4. get nexthop: git clone https://github.com/udhos/nexthop

5. build: \tmp\devel\nexthop\win-build.cmd

6. run: \tmp\devel\nexthop\win-run.cmd

7. telnet to rib daemon: telnet localhost 2001

==EOF==
