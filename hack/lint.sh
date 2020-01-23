#!/usr/bin/env sh

LINT_LOG=lint.log

rm -f ${LINT_LOG}

echo "[Checking FIXME items...]"
git grep -w FIXME | grep -ve 'vendor\|.*\.md\|lint.sh' | tee -a ${LINT_LOG}
[[ ! -s ${LINT_LOG} ]] || exit 1

echo "[Checking lint items...]"
golangci-lint run --no-config --issues-exit-code=1 --deadline=2m --skip-dirs=vendor
