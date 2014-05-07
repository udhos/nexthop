@rem win-goinstall

set DEVEL=c:\tmp\devel
set NEXTHOP=%DEVEL%\nexthop
set GOPATH=%NEXTHOP%

gofmt -s -w %NEXTHOP%\src

@rem build server
go install rib cli rip

@rem eof
