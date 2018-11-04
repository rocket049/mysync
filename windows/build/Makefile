export CGO_ENABLED=0
export GOOS=windows
export GOARCH=386
LDFLAGS=-ldflags -H=windowsgui

all: mysyncd.exe mysync.exe genkey.exe genca.exe

mysyncd.exe:
	go install $(LDFLAGS) mysync/mysyncd
mysync.exe:
	go install mysync/mysync
genkey.exe:
	go install mysync/genkey
genca:
	go install mysync/genca

	
