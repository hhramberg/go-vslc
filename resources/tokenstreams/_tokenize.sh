#!/bin/bash

if [[ $# -lt 2 ]]; then
  echo "Script requires at least two arguments: source directory and output directory."
  exit 1
fi

DIR=$1
OUT=$2
CMP="build/vslc"

if [[ ! -d $DIR ]]; then
  echo "Source directory $DIR does not exist!"
  exit 1
fi

if [[ ! -d $OUT ]]; then
  echo "Output directory $OUT does not exist!"
  exit 1
fi

for e1 in "$DIR"/*.vsl
do
  E="$OUT/$(basename -s .vsl "$e1").txt"
  if [[ ! -f $E ]]; then
    touch "$E"
  fi
  "$CMP" -ts -s "$e1" > "$E"
done