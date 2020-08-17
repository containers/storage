#!/usr/bin/env bash
export PATH=${GOPATH%%:*}/bin:${PATH}
export GIT_VALIDATION=tests/tools/build/git-validation
if [ ! -x "$GIT_VALIDATION" ]; then
	echo git-validation is not installed.
	echo Try installing it with \"make install.tools\"
	exit 1
fi
if test "$TRAVIS" != true ; then
	#GITVALIDATE_EPOCH=":/git-validation epoch"
    GITVALIDATE_EPOCH="9b6484f0058d38a1b85d8b0a3e2ca83684d02e8b"
fi
exec "$GIT_VALIDATION" -q -run DCO,short-subject ${GITVALIDATE_EPOCH:+-range "${GITVALIDATE_EPOCH}""..${GITVALIDATE_TIP:-@}"} ${GITVALIDATE_FLAGS}
