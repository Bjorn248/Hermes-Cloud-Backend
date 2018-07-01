#!/bin/bash

set -e

rm -f device_update device_update.zip
GOOS=linux go build device_update.go
zip device_update.zip ./device_update
rm -f device_update
