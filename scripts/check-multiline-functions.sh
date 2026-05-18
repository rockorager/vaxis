#!/bin/sh
set -eu

one_line_funcs=$(grep -R --include='*.go' -nE '^func .*[{].*[}]$' . --exclude-dir=.git || true)
if [ -n "$one_line_funcs" ]; then
	echo "Go functions must use multiline bodies:"
	echo "$one_line_funcs"
	exit 1
fi
