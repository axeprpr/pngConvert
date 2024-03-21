set GOOS=darwin
go build -o png-convert.MacOS

set GOOS=windows
go build -o png-convert.exe

set GOOS=linux
go build -o png-convert.Linux