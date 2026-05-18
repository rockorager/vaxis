#!/bin/sh
set -eu

files=$(gofumpt -l $(find . -name '*.go' -not -path './.git/*'))
if [ -n "$files" ]; then
	echo "Go files need formatting:"
	echo "$files"
	exit 1
fi
