#!/bin/sh
# Test helper that exits successfully but prints something that is not a response document.
cat >/dev/null
echo "this is not json"
exit 0
