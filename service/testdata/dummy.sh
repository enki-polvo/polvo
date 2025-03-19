#!/bin/bash

while true;
do
    echo "{"eventname": "bashReadline", "source": "eBPF", "timestamp": "2025-03-11T15:29:34+09:00", "log": "A user has entered a command in the bash shell", "metadata": {"Commandline":"echo hello world","PID":191998,"UID":1000,"Username":"shhong"}}"
    sleep 0.01
done