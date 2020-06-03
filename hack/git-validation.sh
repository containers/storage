#!/bin/bash
export PATH=${GOPATH%%:*}/bin:${PATH}
export GIT_VALIDATION=tests/tools/build/git-validation
if [ ! -x "$GIT_VALIDATION" ]; then
	echo git-validation is not installed.
	echo Try installing it with \"make install.tools\"
	exit 1
fi
if test "$TRAVIS" != true ; then
	#GITVALIDATE_EPOCH=":/git-validation epoch"
	GITVALIDATE_EPOCH="0a7c48440c25ec26b4a710c03c957e665f4b2649"
fi
exec "$GIT_VALIDATION" -q -run DCO,short-subject ${GITVALIDATE_EPOCH:+-range "${GITVALIDATE_EPOCH}""..${GITVALIDATE_TIP:-@}"} ${GITVALIDATE_FLAGS}
