// Note that this file has originally been copied from github.com/docker/docker,
// commit 86f080cff091 and been modified afterwards.
//
// Copyright 2012-2017 Docker, Inc.

package opts

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

// Args stores a mapping of keys to a set of multiple values.
type Args struct {
	fields map[string]map[string]bool
}

// KeyValuePair are used to initialize a new Args
type KeyValuePair struct {
	Key   string
	Value string
}

// Arg creates a new KeyValuePair for initializing Args
func Arg(key, value string) KeyValuePair {
	return KeyValuePair{Key: key, Value: value}
}

// NewArgs returns a new Args populated with the initial args
func NewArgs(initialArgs ...KeyValuePair) Args {
	args := Args{fields: map[string]map[string]bool{}}
	for _, arg := range initialArgs {
		args.Add(arg.Key, arg.Value)
	}
	return args
}

// ParseFlag parses a key=value string and adds it to an Args.
//
// Deprecated: Use Args.Add()
func ParseFlag(arg string, prev Args) (Args, error) {
	filters := prev
	if len(arg) == 0 {
		return filters, nil
	}

	name, value, ok := strings.Cut(arg, "=")
	if !ok {
		return filters, ErrBadFormat
	}

	name = strings.ToLower(strings.TrimSpace(name))
	value = strings.TrimSpace(value)

	filters.Add(name, value)

	return filters, nil
}

// ErrBadFormat is an error returned when a filter is not in the form key=value
//
// Deprecated: this error will be removed in a future version
var ErrBadFormat = errors.New("bad format of filter (expected name=value)")

// ToParam encodes the Args as args JSON encoded string
//
// Deprecated: use ToJSON
func ToParam(a Args) (string, error) {
	return ToJSON(a)
}

// MarshalJSON returns a JSON byte representation of the Args
func (args Args) MarshalJSON() ([]byte, error) {
	if len(args.fields) == 0 {
		return []byte{}, nil
	}
	return json.Marshal(args.fields)
}

// ToJSON returns the Args as a JSON encoded string
func ToJSON(a Args) (string, error) {
	if a.Len() == 0 {
		return "", nil
	}
	buf, err := json.Marshal(a)
	return string(buf), err
}

// Get returns the list of values associated with the key
func (args Args) Get(key string) []string {
	values := args.fields[key]
	if values == nil {
		return make([]string, 0)
	}
	slice := make([]string, 0, len(values))
	for key := range values {
		slice = append(slice, key)
	}
	return slice
}

// Add a new value to the set of values
func (args Args) Add(key, value string) {
	if _, ok := args.fields[key]; ok {
		args.fields[key][value] = true
	} else {
		args.fields[key] = map[string]bool{value: true}
	}
}

// Del removes a value from the set
func (args Args) Del(key, value string) {
	if _, ok := args.fields[key]; ok {
		delete(args.fields[key], value)
		if len(args.fields[key]) == 0 {
			delete(args.fields, key)
		}
	}
}

// Len returns the number of keys in the mapping
func (args Args) Len() int {
	return len(args.fields)
}

// MatchKVList returns true if all the pairs in sources exist as key=value
// pairs in the mapping at key, or if there are no values at key.
func (args Args) MatchKVList(key string, sources map[string]string) bool {
	fieldValues := args.fields[key]

	// do not filter if there is no filter set or cannot determine filter
	if len(fieldValues) == 0 {
		return true
	}

	if len(sources) == 0 {
		return false
	}

	for field := range fieldValues {
		key, val, gotVal := strings.Cut(field, "=")

		v, ok := sources[key]
		if !ok {
			return false
		}
		if gotVal && val != v {
			return false
		}
	}

	return true
}

// Match returns true if any of the values at key match the source string
func (args Args) Match(field, source string) bool {
	if args.ExactMatch(field, source) {
		return true
	}

	fieldValues := args.fields[field]
	for name2match := range fieldValues {
		match, err := regexp.MatchString(name2match, source)
		if err != nil {
			continue
		}
		if match {
			return true
		}
	}
	return false
}

// ExactMatch returns true if the source matches exactly one of the values.
func (args Args) ExactMatch(key, source string) bool {
	fieldValues, ok := args.fields[key]
	// do not filter if there is no filter set or cannot determine filter
	if !ok || len(fieldValues) == 0 {
		return true
	}

	// try to match full name value to avoid O(N) regular expression matching
	return fieldValues[source]
}

// UniqueExactMatch returns true if there is only one value and the source
// matches exactly the value.
func (args Args) UniqueExactMatch(key, source string) bool {
	fieldValues := args.fields[key]
	// do not filter if there is no filter set or cannot determine filter
	if len(fieldValues) == 0 {
		return true
	}
	if len(args.fields[key]) != 1 {
		return false
	}

	// try to match full name value to avoid O(N) regular expression matching
	return fieldValues[source]
}

// FuzzyMatch returns true if the source matches exactly one value,  or the
// source has one of the values as a prefix.
func (args Args) FuzzyMatch(key, source string) bool {
	if args.ExactMatch(key, source) {
		return true
	}

	fieldValues := args.fields[key]
	for prefix := range fieldValues {
		if strings.HasPrefix(source, prefix) {
			return true
		}
	}
	return false
}

// Include returns true if the key exists in the mapping
//
// Deprecated: use Contains
func (args Args) Include(field string) bool {
	_, ok := args.fields[field]
	return ok
}

// Contains returns true if the key exists in the mapping
func (args Args) Contains(field string) bool {
	_, ok := args.fields[field]
	return ok
}

type invalidFilterError string

func (e invalidFilterError) Error() string {
	return "Invalid filter '" + string(e) + "'"
}

func (invalidFilterError) InvalidParameter() {}

// Validate compared the set of accepted keys against the keys in the mapping.
// An error is returned if any mapping keys are not in the accepted set.
func (args Args) Validate(accepted map[string]bool) error {
	for name := range args.fields {
		if !accepted[name] {
			return invalidFilterError(name)
		}
	}
	return nil
}

// WalkValues iterates over the list of values for a key in the mapping and calls
// op() for each value. If op returns an error the iteration stops and the
// error is returned.
func (args Args) WalkValues(field string, op func(value string) error) error {
	if _, ok := args.fields[field]; !ok {
		return nil
	}
	for v := range args.fields[field] {
		if err := op(v); err != nil {
			return err
		}
	}
	return nil
}
