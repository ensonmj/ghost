#!/bin/sh

for i in `seq 400 420`; do
    dig +short @114.114.114.114 a r$i-1.googlevideo.com
done | sort -u
