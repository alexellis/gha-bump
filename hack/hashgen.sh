#!/bin/sh

for f in bin/gha-bump*; do shasum -a 256 $f > $f.sha256; done
