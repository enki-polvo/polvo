#!/bin/bash

# SIGINT 시그널 핸들러 정의
trap "echo 'SIGINT received. Exiting...'; exit" SIGINT

while true; do date; sleep 0.01; done

exit 0