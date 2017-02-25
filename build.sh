#!/bin/bash

echo \nRunning hexdump build script

echo \nRemoving old binaries...
rm hexdump/hexdump
rm hexserver/hexserver

echo \nBuilding Docker image...
docker build -t mbgardner/hexdump .

echo \nPushing image to Docker Hub...
docker push mbgardner/hexdump

echo \nAll done\n
