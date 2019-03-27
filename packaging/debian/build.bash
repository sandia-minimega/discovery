#!/bin/bash -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

ROOT=$SCRIPT_DIR/../../

echo BUILDING DISCOVERY...
(cd $ROOT && ./build.bash)
echo DONE BUILDING

echo COPYING FILES...

DST=$SCRIPT_DIR/discovery/opt/discovery
mkdir -p $DST
cp -r $ROOT/bin $DST/
cp -r $ROOT/templates $DST/
mkdir -p $DST/misc
cp -r $ROOT/misc/web $DST/

echo COPIED FILES

echo BUILDING PACKAGE...
(cd $SCRIPT_DIR && fakeroot dpkg-deb -b discovery)
echo DONE
