@rem win-run

set DEVEL=c:\tmp\devel
set NEXTHOP=%DEVEL%\nexthop
set BIN=%NEXTHOP%\bin

start cmd /k %BIN%\rib-old
start cmd /k %BIN%\rib
start cmd /k %BIN%\rip

@rem eof
