@rem win-goinstall

set DEVEL=c:\tmp\devel
set NEXTHOP=%DEVEL%\nexthop
set GOPATH=%NEXTHOP%

go tool fix %NEXTHOP%\src
go tool vet %NEXTHOP%\src\rib-old %NEXTHOP%\src\rib %NEXTHOP%\src\cli %NEXTHOP%\src\rip %NEXTHOP%\src\telnet %NEXTHOP%\src\command
gofmt -s -w %NEXTHOP%\src

@rem build server
go install rib-old rib rip

@rem eof
