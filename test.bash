#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

. $SCRIPT_DIR/env.bash


# testing
echo TESTING
for i in $(find $SCRIPT_DIR/src -mindepth 2 -type d ! -path "*vendor*"); do
    (cd $i && go test -bench .)
done
echo

