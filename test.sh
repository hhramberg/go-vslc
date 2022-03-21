#!/bin/bash

############################
##### Global variables #####
############################

SRC="./src"
BUILD="./build"
VSL="./resources/vsl_typed"
SRC_FILES="$VSL/*.vsl"
VSLC="$BUILD/vslc"
THREADS=4

echo "Typed VSL compiler test"

# Check and verify the existence of folders and files.

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
echo "Testing compiling LLVM IR ..."
for i1 in $SRC_FILES
do
  DST="$BUILD/$(basename "$i1" .vsl).ll"
  echo "Compiling $i1 into LLVM IR"
  touch "$DST"

  # Compile VSL into LLVM ir.
  $VSLC -ll "$i1" > "$DST"

  if [ "$?" ];
  then
    echo -e "\t$i1 compiled successfully into $DST"
  else
    echo -e "\tFailed to compile $i1"
    rm "$DST"
    continue
  fi

  # Assemble LLVM IR, but redirect output to /dev/null.
  llc -o "-" "$DST" > /dev/null 2>&1

  if [ "$?" ];
  then
    echo -e "\t$DST assembled successfully"
  else
    echo -e "\tFailed to assemble $DST"
    rm "$DST"
    continue
  fi

  # Clean up build folder.
  rm "$DST"
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
    continue
  fi

  # Clean up build folder.
  rm "$DST"
  rm "$ELF"
done

echo ""
echo "Testing compiling RISC-V 32-bit assembly ..."
echo ""
echo "Testing compiling ARM 64-bit assembly ..."
echo ""

##############################
##### Multi-thread tests #####
##############################

echo ""
echo "Testing multi-threaded compiler running $THREADS threads"
echo ""
echo "Testing compiling LLVM IR ..."
echo ""
echo "Testing compiling RISC-V 64-bit assembly ..."
echo ""
echo "Testing compiling RISC-V 32-bit assembly ..."
echo ""
echo "Testing compiling ARM 64-bit assembly ..."
echo ""

###################
##### Cleanup #####
###################

echo ""
echo "Cleaning up ..."
echo "Removing VSL compiler $VSLC"
rm "$VSLC"