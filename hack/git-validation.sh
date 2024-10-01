#!/usr/bin/env bash
export GIT_VALIDATION=tests/tools/build/git-validation
if [ ! -x "$GIT_VALIDATION" ]; then
	echo git-validation is not installed.
	echo Try installing it with \"make install.tools\"
	exit 1
fi

EPOCH_TEST_COMMIT=$CIRRUS_BASE_SHA
if [ -z "${EPOCH_TEST_COMMIT}" ]; then
	EPOCH_TEST_COMMIT=$(git merge-base ${DEST_BRANCH:-main} HEAD)
fi
exec "$GIT_VALIDATION" -q -run DCO,short-subject -range "${EPOCH_TEST_COMMIT}..HEAD"
