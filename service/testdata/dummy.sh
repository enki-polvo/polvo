#!/bin/bash

# SIGINT 시그널 핸들러 정의
trap "echo 'SIGINT received. Exiting...'; exit" SIGINT

INDEX=0

while true;
do
    echo "{\"eventname\": \"bashReadline=$INDEX\", \"source\": \"eBPF\", \"timestamp\": \"2025-03-11T15:29:34+09:00\", \"log\": \"A user has entered a command in the bash shell\", \"metadata\": {\"Commandline\":\"echo hello world\",\"PID\":191998,\"UID\":1000,\"Username\":\"shhong\"}}"
    ((INDEX++))
    sleep 0.01
done