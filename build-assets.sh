#!/bin/bash
cd static && polymer build
rm -rf build/bundled/bower*
rm -rf build/bundled/README.md
rm -rf build/bundled/polymer.json
cd ..
go-bindata-assetfs -prefix "static/build/" static/build/bundled/...
