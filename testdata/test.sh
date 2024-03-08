#!/bin/bash
set -eu

arg="$1"

set -x
case "$arg" in
  "add" )
    curl -X POST -F 'bid=100000001' -F 'type=ndlocrv2' -F 'mecabType=chusei-bungo' -F 'localPath=/home/mkunten/Downloads/dev/ocr/0001-000101' http://localhost:1323/api/register
    curl -X POST -F 'bid=200035526' -F 'type=ndlocrv1' -F 'mecabType=chusei-bungo' -F 'localPath=/home/mkunten/Downloads/dev/ocr/200035526_1_3045011_0110-196301' http://localhost:1323/api/register
    ;;
  "bulk" )
    curl -X POST -F 'abortOnError=true' -F 'listcsv=@10.csv' http://localhost:1323/api/bulkRegister
    ;;
  "search" )
    curl -X GET 'http://localhost:1323/api/search?type=ocr&mecabType=chusei_bungo&q=俳諧+和歌'
    ;;
  "recordcount")
    curl -X GET 'http://localhost:1323/api/countRecord'
    ;;
  "search2" )
    curl -X GET 'http://localhost:1323/api/search?type=ocr&q=俳諧+和歌+elevel:OCR'
    ;;
  * )
    echo "\"$1\" not matched"
esac
set +x
