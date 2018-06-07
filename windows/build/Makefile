export CGO_ENABLED=0
export GOOS=windows
export GOARCH=386
LDFLAGS=-ldflags -H=windowsgui

all: mysyncd.exe mysync.exe genkey.exe

mysyncd.exe:~/golang/src/dev/mysyncd/*.go ~/golang/src/dev/mysyncd/*/*.go
	go install $(LDFLAGS) dev/mysyncd
mysync.exe:~/golang/src/dev/mysyncd/mysync/*.go ~/golang/src/dev/mysyncd/*/*.go
	go install dev/mysyncd/mysync
genkey.exe:
	go install dev/mysyncd/genkey

	
