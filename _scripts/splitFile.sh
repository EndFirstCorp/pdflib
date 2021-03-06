#!/bin/sh

#: ./splitFile.sh ~/pdf/1mb/a.pdf ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./splitFile.sh inFile outDir"
    exit 1
fi

f=${1##*/}
f1=${f%.*}
out=$2

#rm -drf $out/*

#set -e

mkdir $out/$f1
cp $1 $out/$f1 

pdflib split -verbose $out/$f1/$f $out/$f1 > $out/$f1/$f1.log
if [ $? -eq 1 ]; then
    echo "split error: $1 -> $out"
    exit $?
else
    echo "split success: $1 -> $out"
    for pdf in $out/$f1/*_?.pdf
    do
        pdflib validate -verbose -mode=relaxed $pdf >> $out/$f1/$f1.log
        if [ $? -eq 1 ]; then
            echo "validation error: $pdf"
            exit $?
        #else
            #echo "validation success: $pdf"
        fi
    done
fi
	

	
