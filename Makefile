all: macos linux_x86_64 linux_mipsle

macos:
	GOOS=darwin go build -ldflags "-w -s"
	/bin/mv puckgo /s3/darwin/
linux_x86_64:
	go build -ldflags "-w -s"
	/bin/mv puckgo /s3/linux/x86_64
linux_mipsle:
	GOARCH=mipsle go build -ldflags "-w -s"
	/bin/mv puckgo /s3/linux/mipsle
