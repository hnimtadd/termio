#!/bin/bash

echo "Testing tab completion..."

# Create some test files to complete
mkdir -p /tmp/termio_test
cd /tmp/termio_test
touch test1.txt test2.txt another_file.txt

echo "=== Files in test directory ==="
ls -la

echo ""
echo "=== Testing in normal zsh ==="
echo "Type 'ls te<TAB>' to see completion"
echo "Expected: should complete to 'test' or show options"

# Test what happens with tab character
echo ""
echo "=== Testing raw tab character ==="
printf "ls te\t" | hexdump -C