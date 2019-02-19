#!/bin/bash

echo Running hexdump build script

echo Removing old binaries...
rm hexdump/hexdump
rm hexserver/hexserver

echo Building hexdump binary...
cd hexdump
CGO_ENABLED=0 go build hexdump.go
cd ..
echo Built hexdump binary

echo Building hexserver binary...
cd hexserver
CGO_ENABLED=0 go build hexserver.go
cd ..
echo Built hexserver binary

echo Building Docker image...
docker build -t mbgardner/hexdump .

echo Pushing image to Docker Hub...
docker push mbgardner/hexdump

echo All done
