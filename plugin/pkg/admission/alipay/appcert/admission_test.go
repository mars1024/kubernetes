package appcert

import "testing"

func TestFetchAppIdentity(t *testing.T) {
	ret, err := fetchAppIdentity("foo")
	if err != nil {
		t.Error(err)
	} else {
		t.Log(ret)
	}
}