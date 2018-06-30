#!/bin/bash

set -e

rm -f user_registration user_registration.zip
GOOS=linux go build user_registration.go
zip user_registration.zip ./user_registration
rm -f user_registration
