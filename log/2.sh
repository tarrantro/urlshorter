#!/bin/bash

FILE=$1

if [[ ! -f ${FILE} ]]; then
    echo "file not exist!"
    exit 255
fi

echo "10 IP access most to server:" 
for i in $(awk '$4 >= "[17/May/2015:00:00:00" && $4 <= "[20/May/2015:23:59:59" { print $0 }' ${FILE}|awk '{print $1}'|sort -n|uniq -c|sort -rn|head -n 10|awk '{print $2}')
do
    echo $i
done