#!/bin/bash -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

ROOT_DIR=$SCRIPT_DIR/../../

echo BUILDING DISCOVERY...
(cd $ROOT_DIR/scripts && ./build.bash)
echo DONE BUILDING

echo COPYING FILES...

DST=$SCRIPT_DIR/discovery/opt/discovery
mkdir -p $DST
cp -r $ROOT_DIR/bin $DST/
cp -r $ROOT_DIR/templates $DST/
mkdir -p $DST/misc
cp -r $ROOT_DIR/misc/web $DST/misc/

echo COPIED FILES

echo BUILDING PACKAGE...
(cd $SCRIPT_DIR && fakeroot dpkg-deb -b discovery)
echo DONE
