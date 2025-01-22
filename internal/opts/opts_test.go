package opts_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/containers/storage/internal/opts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateIPAddress(t *testing.T) {
	if ret, err := opts.ValidateIPAddress(`1.2.3.4`); err != nil || ret == "" {
		t.Fatalf("ValidateIPAddress(`1.2.3.4`) got %s %s", ret, err)
	}

	if ret, err := opts.ValidateIPAddress(`127.0.0.1`); err != nil || ret == "" {
		t.Fatalf("ValidateIPAddress(`127.0.0.1`) got %s %s", ret, err)
	}

	if ret, err := opts.ValidateIPAddress(`::1`); err != nil || ret == "" {
		t.Fatalf("ValidateIPAddress(`::1`) got %s %s", ret, err)
	}

	if ret, err := opts.ValidateIPAddress(`127`); err == nil || ret != "" {
		t.Fatalf("ValidateIPAddress(`127`) got %s %s", ret, err)
	}

	if ret, err := opts.ValidateIPAddress(`random invalid string`); err == nil || ret != "" {
		t.Fatalf("ValidateIPAddress(`random invalid string`) got %s %s", ret, err)
	}

}

func TestMapOpts(t *testing.T) {
	tmpMap := make(map[string]string)
	o := opts.NewMapOpts(tmpMap, logOptsValidator)
	err := o.Set("max-size=1")
	require.NoError(t, err)
	if o.String() != "map[max-size:1]" {
		t.Errorf("%s != [map[max-size:1]", o.String())
	}

	err = o.Set("max-file=2")
	require.NoError(t, err)
	if len(tmpMap) != 2 {
		t.Errorf("map length %d != 2", len(tmpMap))
	}

	if tmpMap["max-file"] != "2" {
		t.Errorf("max-file = %s != 2", tmpMap["max-file"])
	}

	if tmpMap["max-size"] != "1" {
		t.Errorf("max-size = %s != 1", tmpMap["max-size"])
	}
	if o.Set("dummy-val=3") == nil {
		t.Errorf("validator is not being called")
	}
}

func TestListOptsWithoutValidator(t *testing.T) {
	o := opts.NewListOpts(nil)
	err := o.Set("foo")
	require.NoError(t, err)
	if o.String() != "[foo]" {
		t.Errorf("%s != [foo]", o.String())
	}
	err = o.Set("bar")
	require.NoError(t, err)
	if o.Len() != 2 {
		t.Errorf("%d != 2", o.Len())
	}
	err = o.Set("bar")
	require.NoError(t, err)
	if o.Len() != 3 {
		t.Errorf("%d != 3", o.Len())
	}
	if !o.Get("bar") {
		t.Error("o.Get(\"bar\") == false")
	}
	if o.Get("baz") {
		t.Error("o.Get(\"baz\") == true")
	}
	o.Delete("foo")
	if o.String() != "[bar bar]" {
		t.Errorf("%s != [bar bar]", o.String())
	}
	listOpts := o.GetAll()
	if len(listOpts) != 2 || listOpts[0] != "bar" || listOpts[1] != "bar" {
		t.Errorf("Expected [[bar bar]], got [%v]", listOpts)
	}
	mapListOpts := o.GetMap()
	if len(mapListOpts) != 1 {
		t.Errorf("Expected [map[bar:{}]], got [%v]", mapListOpts)
	}

}

func TestListOptsWithValidator(t *testing.T) {
	// Reusing logOptsvalidator (used by MapOpts)
	o := opts.NewListOpts(logOptsValidator)
	err := o.Set("foo")
	assert.EqualError(t, err, "invalid key foo")
	if o.String() != "[]" {
		t.Errorf("%s != []", o.String())
	}
	err = o.Set("foo=bar")
	assert.EqualError(t, err, "invalid key foo")
	if o.String() != "[]" {
		t.Errorf("%s != []", o.String())
	}
	err = o.Set("max-file=2")
	require.NoError(t, err)
	if o.Len() != 1 {
		t.Errorf("%d != 1", o.Len())
	}
	if !o.Get("max-file=2") {
		t.Error("o.Get(\"max-file=2\") == false")
	}
	if o.Get("baz") {
		t.Error("o.Get(\"baz\") == true")
	}
	o.Delete("max-file=2")
	if o.String() != "[]" {
		t.Errorf("%s != []", o.String())
	}
}

func TestValidateLabel(t *testing.T) {
	if _, err := opts.ValidateLabel("label"); err == nil || err.Error() != "bad attribute format: label" {
		t.Fatalf("Expected an error [bad attribute format: label], go %v", err)
	}
	if actual, err := opts.ValidateLabel("key1=value1"); err != nil || actual != "key1=value1" {
		t.Fatalf("Expected [key1=value1], got [%v,%v]", actual, err)
	}
	// Validate it's working with more than one =
	if actual, err := opts.ValidateLabel("key1=value1=value2"); err != nil {
		t.Fatalf("Expected [key1=value1=value2], got [%v,%v]", actual, err)
	}
	// Validate it's working with one more
	if actual, err := opts.ValidateLabel("key1=value1=value2=value3"); err != nil {
		t.Fatalf("Expected [key1=value1=value2=value2], got [%v,%v]", actual, err)
	}
}

func logOptsValidator(val string) (string, error) {
	allowedKeys := map[string]string{"max-size": "1", "max-file": "2"}
	vals := strings.Split(val, "=")
	if allowedKeys[vals[0]] != "" {
		return val, nil
	}
	return "", fmt.Errorf("invalid key %s", vals[0])
}

func TestNamedListOpts(t *testing.T) {
	var v []string
	o := opts.NewNamedListOptsRef("foo-name", &v, nil)

	err := o.Set("foo")
	require.NoError(t, err)
	if o.String() != "[foo]" {
		t.Errorf("%s != [foo]", o.String())
	}
	if o.Name() != "foo-name" {
		t.Errorf("%s != foo-name", o.Name())
	}
	if len(v) != 1 {
		t.Errorf("expected foo to be in the values, got %v", v)
	}
}

func TestNamedMapOpts(t *testing.T) {
	tmpMap := make(map[string]string)
	o := opts.NewNamedMapOpts("max-name", tmpMap, nil)

	err := o.Set("max-size=1")
	require.NoError(t, err)
	if o.String() != "map[max-size:1]" {
		t.Errorf("%s != [map[max-size:1]", o.String())
	}
	if o.Name() != "max-name" {
		t.Errorf("%s != max-name", o.Name())
	}
	if _, exist := tmpMap["max-size"]; !exist {
		t.Errorf("expected map-size to be in the values, got %v", tmpMap)
	}
}
