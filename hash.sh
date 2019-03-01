#!/bin/bash
INCR=0
while :
do
    curl -X POST 127.0.0.1:8080/hashes -d '{"hash":"c3ab8ff13720e8ad97dd39466b3c8974e592c2fa383d4a3960714caef0c4f2"}' -H "Content-Type: application/json"
    INCR=$((INCR + 1))
    printf "\n$INCR\n"
    sleep 1
done
