#!/bin/bash

FILE=$1

if [[ ! -f ${FILE} ]]; then
    echo "file not exist!"
    exit 255
fi

if [[ -f "./tmp" ]]; then
    rm -f ./tmp
fi

cat ${FILE}|awk '{print $1}'| while read line
do
    country=$(geoiplookup $line|awk '{print $4}'| awk -F, '{print $1}')
    echo $country >> ./tmp
done


echo "10 geo location access most to server:" 
cat "./tmp"|sort -n|uniq -c|sort -rn|head -n 10