#!/bin/bash

############################
##### Global variables #####
############################

SRC="./src"
BUILD="./build"
VSL="./resources/vsl_typed"
SRC_FILES="$VSL/*.vsl"
VSLC="$BUILD/vslc"
GCC="aarch64-linux-gnu-gcc"
QEMU="qemu-aarch64"
TARGET="aarch64"
declare -i TESTS=0
declare -i FAILED=0
declare -i PASSED=0

echo "Typed VSL compiler test"

###############################################################
##### Check and verify the existence of folders and files #####
###############################################################

if $GCC --version &> /dev/null;
then
  echo "Found GCC for target $TARGET"
else
  echo "Error: $GCC is not installed"
  exit 1
fi

if $QEMU --version &> /dev/null;
then
  echo "Found QEMU for $TARGET"
else
  echo "Error: $QEMU is not installed"
  exit 1
fi

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

################################################
##### Test compiled program with arguments #####
################################################

echo ""
echo "Testing single threaded compiler"
echo ""
echo "Testing compiling into $TARGET assembler using LIR intermediate representation"
echo "Running the compiled program with arguments provided in the first commented line of every source code file"
for i1 in $SRC_FILES
do
  echo ""
  TESTS=$((TESTS + 1))
  DST="$BUILD/$(basename "$i1" .vsl)"
  echo -n "Compiling $i1 using LIR IR and backend into $TARGET assembler ... "

  # Compile VSL into executable using LIR IR and backend.
  if $VSLC -arch "$TARGET" -o "$DST".s "$i1";
  then
    echo "SUCCESS!"
  else
    FAILED=$((FAILED+1))
    echo "FAILED!"
    continue
  fi

  echo -n "Compiling $i1 into $TARGET binary using GCC ... "
  if aarch64-linux-gnu-gcc -static -o "$DST" "$DST".s;
  then
    echo "SUCCESS!"
  else
    FAILED=$((FAILED+1))
    echo "FAILED!"
    continue
  fi

  LINE=$(sed -n "1p" "$i1")
  if [[ "$LINE" == //* ]];
  then
    ARGS=$(echo "$LINE" | grep -oiE '[0-9]+|([0-9]+.[0-9]+)' | xargs)
    echo -n "Executing $DST with arguments: [$ARGS] ... "
    if echo "$ARGS" | xargs qemu-aarch64 "$DST" &> /dev/null;
    then
      echo "SUCCESS!"
      PASSED=$((PASSED + 1))
    else
      echo "FAILED!"
      FAILED=$((FAILED + 1))
    fi

  else
    FAILED=$((FAILED + 1))
    echo "$i1's first line isn't a comment"
    continue
  fi
done



echo ""
echo "Testing multithreading compiler"
echo ""
echo "Testing compiling into $TARGET assembler using LIR intermediate representation"
echo "Running the compiled program with arguments provided in the first commented line of every source code file"
echo ""
for i1 in $SRC_FILES
do
  LINE=$(sed -n "1p" "$i1")
  if [[ "$LINE" == //* ]];
  then
    ARGS=$(echo "$LINE" | grep -oiE '[0-9]+|([0-9]+.[0-9]+)' | xargs)
  else
    FAILED=$((FAILED + 1))
    echo "$i1's first line isn't a comment"
    continue
  fi
  for i2 in 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16
  do
    TESTS=$((TESTS + 1))
    DST="$BUILD/$(basename "$i1" .vsl)"
    echo -n "Compiling $i1 using LIR IR and backend with up to $i2 threads into $TARGET assembler ... "

    # Compile VSL into executable using LIR IR and backend.
    if $VSLC -t "$i2" -arch "$TARGET" -o "$DST".s "$i1";
    then
      echo "SUCCESS!"
    else
      FAILED=$((FAILED+1))
      echo "FAILED!"
      continue
    fi

    echo -n "Compiling $i1 into $TARGET binary using GCC ... "
    if aarch64-linux-gnu-gcc -static -o "$DST" "$DST".s;
    then
      echo "SUCCESS!"
    else
      FAILED=$((FAILED+1))
      echo "FAILED!"
      continue
    fi

    echo -n "Executing $DST with arguments: [$ARGS] ... "
    if echo "$ARGS" | xargs qemu-aarch64 "$DST" &> /dev/null;
    then
      echo "SUCCESS!"
      PASSED=$((PASSED + 1))
    else
      echo "FAILED!"
      FAILED=$((FAILED + 1))
    fi

  done
done

echo ""
echo "Tests complete! $TESTS tests were run."
echo "$PASSED tests passed, $FAILED tests failed."

if [ $PASSED == $TESTS ];
then
  echo "All $TESTS test passed."
fi

###################
##### Cleanup #####
###################

echo "Cleaning up ..."
echo "Removing VSL compiler at $VSLC"
rm "$VSLC"

exit 0