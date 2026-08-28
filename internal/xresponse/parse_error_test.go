package xresponse

import (
	"errors"
	"testing"
)

func TestIsRequestParseError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{errors.New(`field "page" is not fully set`), true},
		{errors.New(`field "id" is not set`), true},
		{errors.New(`type mismatch for field "id"`), true},
		{errors.New("invalid character 'x' looking for beginning of value"), true},
		{errors.New("json: cannot unmarshal string into Go value of type int64"), true},
		{errors.New("wrong number range setting"), true}, // range=[0:150] 传了 999
		{errors.New("unexpected EOF"), true},
		{errors.New("record not found"), false},
		{errors.New("dial tcp: connection refused"), false},
	}
	for _, c := range cases {
		if got := isRequestParseError(c.err); got != c.want {
			t.Errorf("isRequestParseError(%q) = %v, want %v", c.err, got, c.want)
		}
	}
}
