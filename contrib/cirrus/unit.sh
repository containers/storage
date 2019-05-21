#!/bin/bash

set -e

source $(dirname $0)/lib.sh

cd $SCRIPT_BASE
./lib.sh.t

cd $GOSRC
/bin/true  # STUB: Add call to other unittests
