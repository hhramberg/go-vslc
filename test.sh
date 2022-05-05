#!/bin/bash

############################
##### Global variables #####
############################

SRC="./src"
BUILD="./build"
VSL="./resources/vsl_typed"
SRC_FILES="$VSL/*.vsl"
VSLC="$BUILD/vslc"
TARGET="aarch64"
declare -i TESTS=0
declare -i FAILED=0
declare -i PASSED=0

echo "Typed VSL compiler test"

###############################################################
##### Check and verify the existence of folders and files #####
###############################################################

if [ ! -d "$VSL" ];
then
  echo "Error: directory $VSL does not exist"
  exit 1
fi

if [ ! -d "$BUILD" ];
then
  echo "Error: directory $BUILD does not exist"
  exit 1
fi

if [ ! -d "$SRC" ];
then
  echo "Error: directory $SRC does not exist"
  exit 1
fi

if [ ! -f "$SRC/main.go" ];
then
  echo "Error: source code file $SRC/main.go does not exist"
  exit 1
fi

if go build -o "$VSLC" "$SRC/main.go" ;
then
  echo "Compiled VSL compiler and put it in $VSLC"
else
  echo "Compilation failed"
  exit 1
fi

###############################
##### Single thread tests #####
###############################

echo ""
echo "Testing single threaded compiler"
echo ""
echo "Testing compiling into $TARGET assembler using LIR intermediate representation"
echo ""
for i1 in $SRC_FILES
do
  TESTS=$((TESTS + 1))
  DST="$BUILD/$(basename "$i1" .vsl)"
  echo -n "Compiling $i1 ... "

  # Compile VSL into executable using LLVM backend.

  if $VSLC -arch "$TARGET" -o "$DST" "$i1";
  then
    echo "SUCCESS!"
    PASSED=$((PASSED + 1))
  else
    echo "FAILED!"
    echo "$ERR"
    FAILED=$((FAILED + 1))
    continue
  fi
  rm "$DST"
done

echo ""
echo "Tests complete! $TESTS tests were run."
echo "$PASSED tests passed, $FAILED tests failed."

if [ $PASSED == $TESTS ];
then
  echo "All $TESTS test passed."
fi

################################
##### Multithreading tests #####
################################

echo ""
echo "Testing multithreading threaded compiler"
echo ""
echo "Testing compiling into $TARGET assembler using LIR intermediate representation"
echo ""
for i1 in $SRC_FILES
do
  # Testing parallel compiling running [2, 16] threads i parallel.
  DST="$BUILD/$(basename "$i1" .vsl)"
  for i2 in 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16
  do
    TESTS=$((TESTS + 1))
    echo -n "Compiling $i1 with up to $i2 threads ... "

    # Compile VSL into executable using LLVM backend.

    if $VSLC -t "$i2" -arch "$TARGET" -o "$DST" "$i1";
    then
      echo "SUCCESS!"
      PASSED=$((PASSED + 1))
    else
      echo "FAILED!"
      echo "$ERR"
      FAILED=$((FAILED + 1))
      break
    fi
  done
  rm "$DST"
done

#########################
##### LLVM IR TESTS #####
#########################

echo ""
echo "Testing single threaded compiler"
echo ""
echo "Testing compiling using LLVM targeting $TARGET"
echo ""
for i1 in $SRC_FILES
do
  TESTS=$((TESTS + 1))
  DST="$BUILD/$(basename "$i1" .vsl)"
  echo -n "Compiling $i1 using LLVM IR and backend ... "

  # Compile VSL into executable using LLVM backend.
  if $VSLC -ll -arch "$TARGET" -o "$DST" "$i1";
  then
    PASSED=$((PASSED+1))
    echo "SUCCESS!"
  else
    FAILED=$((FAILED+1))
    echo "FAILED!"
    continue
  fi

  # Clean up build folder.
  rm "$DST"
done

echo ""
echo "Tests complete! $TESTS tests were run."
echo "$PASSED tests passed, $FAILED tests failed."

if [ $PASSED == $TESTS ];
then
  echo "All $TESTS test passed."
fi

################################
##### Multithreading tests #####
################################

#echo ""
#echo "Testing multithreading threaded compiler"
#echo ""
#echo "Testing compiling using LLVM targeting $TARGET"
#echo ""
#for i1 in $SRC_FILES
#do
#  # Testing parallel compiling running [2, 16] threads i parallel.
#  for i2 in 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16
#  do
#    TESTS=$((TESTS + 1))
#    DST="$BUILD/$(basename "$i1" .vsl)"
#    echo -n "Compiling $i1 using LLVM IR and backend with up to $i2 threads ... "
#
#    # Compile VSL into executable using LLVM backend.
#    if $VSLC -t "$i2" -ll -arch "$TARGET" -o "$DST" "$i1";
#    then
#      PASSED=$((PASSED+1))
#      echo "SUCCESS!"
#    else
#      FAILED=$((FAILED+1))
#      echo "FAILED!"
#      break
#    fi
#
#    # Clean up build folder.
#    rm "$DST"
#  done
#done
#
#echo ""
#echo "Tests complete! $TESTS tests were run."
#echo "$PASSED tests passed, $FAILED tests failed."
#
#if [ $PASSED == $TESTS ];
#then
#  echo "All $TESTS test passed."
#fi

###################
##### Cleanup #####
###################

echo "Cleaning up ..."
echo "Removing VSL compiler at $VSLC"
rm "$VSLC"

exit 0