package apiserver

import (
	"testing"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		serverAddr       string
		expectedHost     string
		expectedHostName string
		expectedErr      bool
	}{
		{
			serverAddr:       "https://test.me:80",
			expectedHost:     "test.me:80",
			expectedHostName: "test.me",
		},
		{
			serverAddr:       "https://test.me:80/",
			expectedHost:     "test.me:80",
			expectedHostName: "test.me",
		},
		{
			serverAddr:       "https://test.me/",
			expectedHost:     "test.me",
			expectedHostName: "test.me",
		},
		{
			serverAddr:       "test.me:8080",
			expectedHost:     "test.me:8080",
			expectedHostName: "test.me",
		},
	}

	for idx, tc := range testCases {
		gotHost, gotHostName, err := parseAKEApiService(tc.serverAddr)
		if err != nil {
			if !tc.expectedErr {
				t.Errorf("[%d] unexpected error: %v", idx, err)
			}
		} else {
			if tc.expectedErr {
				t.Errorf("[%d] expected error, but got nil", idx)
			} else {
				if gotHost != tc.expectedHost {
					t.Errorf("[%d] host not equal: expected %q, but got %q", idx, tc.expectedHost, gotHost)
				}
				if gotHostName != tc.expectedHostName {
					t.Errorf("[%d] hostName not equal: expected %q, but got %q", idx, tc.expectedHostName, gotHostName)
				}
			}
		}
	}
}
