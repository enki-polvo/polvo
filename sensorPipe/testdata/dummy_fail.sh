#!/bin/bash

# SIGINT 시그널 핸들러 정의
trap "echo 'SIGINT received. Exiting...'; exit" SIGINT

echo "This is a dummy script that fails"

INDEX=0

while [ $INDEX -le 100 ]
do
    date
    ((INDEX++))
    sleep 0.01
done

exit 255