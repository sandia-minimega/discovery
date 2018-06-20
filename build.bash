#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

. $SCRIPT_DIR/env.bash

# build packages
echo "BUILD PACKAGES (linux)"
for i in $(ls $SCRIPT_DIR/src/cmds); do
	echo $i
    (cd $SCRIPT_DIR/src/cmds/$i && go install)
done
echo

