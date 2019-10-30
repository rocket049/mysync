#!/bin/sh
export CGO_ENABLED=0
export GOOS=windows
export GOARCH=386
LDFLAGS='-ldflags -H=windowsgui'
go install ${LDFLAGS} mysync/mysyncd
go install mysync/mysync
go install mysync/genkey
go install mysync/genca
