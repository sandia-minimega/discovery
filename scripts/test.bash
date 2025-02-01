#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$( cd ${SCRIPT_DIR}/.. && pwd )"

. $SCRIPT_DIR/env.bash

DIRECTORY_ARRAY=("$ROOT_DIR/cmd $ROOT_DIR/pkg")

# testing
echo TESTING
for i in ${DIRECTORY_ARRAY[@]}; do
    (cd $i && go test -bench .)
done
echo
