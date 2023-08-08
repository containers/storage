package toc

import (
	"testing"
)

func TestGetTOCDigest(t *testing.T) {
	t.Run("ValidTOCDigestAnnotation", func(t *testing.T) {
		expectedDigest := "sha256:8bc94b65d0b3ae8998cc0405a424ee7c3a04c72996f99eda9670374832dc9667"
		annotations := map[string]string{
			tocJSONDigestAnnotation: expectedDigest,
		}

		digestPtr, err := GetTOCDigest(annotations)
		if err != nil {
			t.Error(err)
		}
		if digestPtr == nil {
			t.Errorf("Expected a non-nil digest pointer")
		} else if digestPtr.String() != expectedDigest {
			t.Errorf("Expected digest %s, but got %s", expectedDigest, digestPtr.String())
		}
	})

	t.Run("InvalidTOCDigestAnnotation", func(t *testing.T) {
		annotations := map[string]string{
			tocJSONDigestAnnotation: "invalid-checksum",
		}

		_, err := GetTOCDigest(annotations)
		if err == nil {
			t.Fatal("Expected error")
		}
	})

	t.Run("NoValidAnnotations", func(t *testing.T) {
		annotations := map[string]string{}

		digestPtr, err := GetTOCDigest(annotations)
		if err != nil {
			t.Error(err)
		}
		if digestPtr != nil {
			t.Errorf("Expected nil digest pointer, but got %s", digestPtr.String())
		}
	})
}
