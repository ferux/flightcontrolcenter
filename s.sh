#!/bin/bash
set -e

result="`curl -o /dev/null -sw '%{http_code}\n' https://fcc.loyso.art/api/v2/info`"
[ $result -eq 200 ] && echo 'okay' && exit 0 || echo ${result} && exit 1
