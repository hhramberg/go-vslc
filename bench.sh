#!/bin/bash

############################
##### Global variables #####
############################

SRC="./src"
DST=$(echo "`pwd`/doc/bench.txt")

cd "$SRC" || exit 1

if [ ! -f "DST" ];
then
  touch "$DST"
  echo "Created $DST"
fi

echo "[`date +"%Y-%m-%d %T"`]: Starting benchmarking, this will take some time"
go test -bench ../. &> "$DST"
echo "[`date +"%Y-%m-%d %T"`]: Benchmarking finished!"
echo "Results were written to $DST"

cd .. || exit 1

exit 0