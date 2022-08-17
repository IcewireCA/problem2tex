#!/bin/sh
# to run this shell script use: bash buildExecutables.sh
#
# Script is used to build and zip binaries/problem2tex versions
date=_220724
env GOOS=windows GOARCH=amd64 go build -o binaries/problem2tex.exe
tar -czf binaries/problem2texWin64$date.tar.gz binaries/problem2tex.exe
rm binaries/problem2tex.exe
env GOOS=linux GOARCH=amd64 go build -o binaries/problem2tex
tar -czf binaries/problem2texLinux64$date.tar.gz binaries/problem2tex
rm binaries/problem2tex
env GOOS=darwin GOARCH=amd64 go build -o binaries/problem2tex
tar -czf binaries/problem2texMacOS64$date.tar.gz binaries/problem2tex
rm binaries/problem2tex

