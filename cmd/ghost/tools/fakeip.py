#!/usr/bin/python
# -*- coding: utf-8 -*-
"""
find fake ip
"""

from __future__ import print_function
import random
from subprocess import Popen, PIPE


def fakeip():
    "find fake ip"
    result = set()
    start = random.randint(0, 1000000)
    for i in range(start, start + 20):
        p = Popen((('dig +short @114.114.114.114 a r%d-1.googlevideo.com') % i).split(),
                  stdin=PIPE, stdout=PIPE, stderr=PIPE, close_fds=True)
        output = p.stdout.read()
        if output:
            result.add(output.strip())

    return result

if __name__ == '__main__':
    for r in fakeip():
        print(r)
