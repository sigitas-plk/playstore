package playstore

import (
	"strings"
	"testing"
)

func TestFileSha256(t *testing.T) {

	t.Run("should return expected sha hash", func(t *testing.T) {
		data := "testinputdata"
		expected := "b5a7e3884f760b197b8e9df98bab2d35333943bc7439e0bba00ed87213a44f43"
		r := strings.NewReader(data)

		actual, err := fileSha256(r)
		if err != nil {
			t.Fatal(err)
		}

		if actual != expected {
			t.Errorf("want '%s' hash, got '%s'", expected, actual)
		}
	})
}
