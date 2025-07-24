#!/bin/bash
if [ -z "$1" ]; then
  test_path="."
elif [ "$1" == "integration" ]; then
  test_path="./integration"
elif [ "$1" == "unit" ]; then
  test_path="./unit"
else
  echo "Invalid argument. Use 'integration' or 'unit'."
  exit 1
fi

go test -v ${test_path} | sed ''/PASS/s//$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$(printf "\033[31mFAIL\033[0m")/''
