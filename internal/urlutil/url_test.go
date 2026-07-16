package urlutil

import "testing"

func TestValidateHTTP(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
		ok    bool
	}{
		{name: "https", value: "https://example.com/watch?v=1", ok: true},
		{name: "http", value: "http://example.com/song", ok: true},
		{name: "missing host", value: "https://"},
		{name: "file", value: "file:///etc/passwd"},
		{name: "control", value: "https://example.com/a\nb"},
		{name: "userinfo", value: "https://user:pass@example.com/song"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateHTTP(test.value)
			if (err == nil) != test.ok {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
