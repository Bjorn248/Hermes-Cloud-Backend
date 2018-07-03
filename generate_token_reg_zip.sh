#!/bin/bash

set -e

rm -f token_generate token_generate.zip
GOOS=linux go build token_generate.go
zip token_generate.zip ./token_generate
rm -f token_generate
