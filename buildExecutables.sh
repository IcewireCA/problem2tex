#!/bin/sh
# to run this shell script use: bash buildExecutables.sh
#
# Script is used to build and zip binaries/problem2tex versions
date=_210302
env GOOS=windows GOARCH=amd64 go build -o binaries/problem2tex.exe
zip -r binaries/problem2texWin64$date.zip binaries/problem2tex.exe
rm binaries/problem2tex.exe
env GOOS=linux GOARCH=amd64 go build -o binaries/problem2tex
zip -r binaries/problem2texLinux64$date.zip binaries/problem2tex
rm binaries/problem2tex
env GOOS=darwin GOARCH=amd64 go build -o binaries/problem2tex
zip -r binaries/problem2texMacOS64$date.zip binaries/problem2tex
rm binaries/problem2tex

