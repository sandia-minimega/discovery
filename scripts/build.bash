#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$( cd ${SCRIPT_DIR}/.. && pwd)"

. $SCRIPT_DIR/env.bash

DIRECTORY_ARRAY=("$ROOT_DIR/cmd")

# build packages
echo "BUILD PACKAGES (linux)"
for i in $ROOT_DIR/cmd/*; do
    echo $(basename $i)
    (cd $i && go install)
done
echo
