#!/bin/bash

FILE=$1

if [[ ! -f ${FILE} ]]; then
    echo "file not exist!"
    exit 255
fi

echo "http request number:" $(awk '{print $8}' ${FILE}|grep -i '^HTTP'|wc -l)