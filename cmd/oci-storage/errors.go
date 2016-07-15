package main

import "errors"

var (
	missingArgError          = errors.New("required argument not provided")
	noMatchingContainerError = errors.New("no container by that name")
)
