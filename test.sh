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

echo "Testing single threaded compiler"
echo ""
echo "Testing compiling into LIR intermediate representation to verify compiler frontend and intermediate stage"
echo ""
for i1 in $SRC_FILES
do
  TESTS=$((TESTS + 1))
  DST="$BUILD/$(basename "$i1" .vsl)"
  echo -n "Compiling $i1 ... "

  # Compile VSL into executable using LLVM backend.
   # | read ERR # > /dev/null # Redirect errors to /dev/null.

  if $VSLC "$i1";
  then
    echo "SUCCESS!"
    PASSED=$((PASSED + 1))
  else
    echo "FAILED!"
    echo "$ERR"
    FAILED=$((FAILED + 1))
    continue
  fi
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

# Reset status counters.
TESTS=$((0))
PASSED=$((0))
FAILED=$((0))

echo ""
echo "Testing single threaded compiler"
echo ""
echo "Testing compiling into LIR intermediate representation to verify compiler frontend and intermediate stage"
echo ""
for i1 in $SRC_FILES
do
  # Testing parallel compiling running [2, 16] threads i parallel.
  for i2 in 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16
  do
    TESTS=$((TESTS + 1))
    DST="$BUILD/$(basename "$i1" .vsl)"
    echo -n "Compiling $i1 with up to $i2 threads ... "

    # Compile VSL into executable using LLVM backend.

    if $VSLC -t "$i2" "$i1";
    then
      echo "SUCCESS!"
      PASSED=$((PASSED + 1))
    else
      echo "FAILED!"
      echo "$ERR"
      FAILED=$((FAILED + 1))
      continue
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

############################
##### DELETE TEMP TEST #####
############################

echo "Testing single threaded compiler"
echo ""
echo "Testing compiling using LLVM targeting $TARGET"
for i1 in $SRC_FILES
do
  DST="$BUILD/$(basename "$i1" .vsl)"
  echo "Compiling $i1 using LLVM IR and backend"

  # Compile VSL into executable using LLVM backend.
  $VSLC -ll -target aarch64 -o "$DST" "$i1" # > /dev/null # Redirect errors to /dev/null.

  if [ "$?" ];
  then
    echo -e "\t$i1 compiled successfully into $DST"
  else
    echo -e "\tFailed to compile $i1"
    rm "$DST"
    FAILED=$((FAILED + 1))
    continue
  fi

  # Clean up build folder.
  rm "$DST"
  PASSED=$((PASSED + 1))
done

echo ""
echo "Testing compiling RISC-V 64-bit assembly ..."
echo ""
for i1 in $SRC_FILES
do
  DST="$BUILD/$(basename "$i1" .vsl).s"
  echo "Compiling $i1 into RISC-V 64-bit assembly ..."
  touch "$DST"

  # Compile VSL into LLVM ir.
  $VSLC -target "riscv64" -o "$DST" "$i1"

  if [ "$?" ];
  then
    echo -e "\t$i1 compiled successfully into $DST"
  else
    echo -e "\tFailed to compile $i1"
    rm "$DST"
    FAILED=$((FAILED + 1))
    continue
  fi

  # Assemble RISC-V 64-bit assembly into ELF.
  echo "Assembling RISC-V 64-bit assembly into ELF executable ..."
  ELF=$BUILD/$(basename "$i1" .vsl)
  riscv64-linux-gnu-as -o "$ELF" "$DST"

  if [ "$?" ];
  then
    echo -e "\t$DST assembled successfully into $ELF"
  else
    echo -e "\tFailed to assemble $DST"
    rm "$DST"
    FAILED=$((FAILED + 1))
    continue
  fi

  # Clean up build folder.
  rm "$DST"
  rm "$ELF"
  PASSED=$((PASSED + 1))
done

echo ""
echo "Testing compiling RISC-V 32-bit assembly ..."
echo ""
echo "Testing compiling ARM 64-bit assembly ..."
echo ""

##############################
##### Multi-thread tests #####
##############################

# Test from 2 to 16 threads.
for i1 in 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16
do
  echo ""
  echo "Testing multi-threaded compiler running $i1 threads"
  echo ""
  echo "Testing compiling using LLVM IR and backend using $i1 threads ..."
  echo ""
  echo "Testing compiling RISC-V 64-bit assembly using $i1 threads ..."
  echo ""
  echo "Testing compiling RISC-V 32-bit assembly using $i1 threads ..."
  echo ""
  echo "Testing compiling ARM 64-bit assembly using $i1 threads ..."
  echo ""
done

##############################
##### Print test results #####
##############################

echo ""
echo "Tests complete!"
echo "$PASSED tests passed, $FAILED tests failed."

###################
##### Cleanup #####
###################

echo "Cleaning up ..."
echo "Removing VSL compiler $VSLC"
rm "$VSLC"