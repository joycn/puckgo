all: macos linux

macos:
	GOOS=darwin go build -ldflags "-w -s"
	/bin/cp puckgo /s3/darwin/
linux:
	go build -ldflags "-w -s"
	/bin/cp puckgo /s3/linux/
