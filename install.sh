#!/bin/bash
set -e
set -x

go build
chmod a+x ./dirtitle
cp ./dirtitle ~/bin/

