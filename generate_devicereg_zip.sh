#!/bin/bash

set -e

rm -f device_registration device_registration.zip
GOOS=linux go build device_registration.go
zip device_registration.zip ./device_registration
rm -f device_registration
