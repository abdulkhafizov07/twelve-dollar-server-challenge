#!/usr/bin/env bash

GOMAXPROCS=1 exec "$(dirname "$0")/server"

