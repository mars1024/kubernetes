package apiserver

import (
	"testing"
)

func TestParseAKEApiService(t *testing.T) {
	testCases := []struct {
		serverAddr       string
		expectedScheme   string
		expectedHost     string
		expectedHostName string
		expectedErr      bool
	}{
		{
			serverAddr:       "https://test.me:80",
			expectedScheme:   "https",
			expectedHost:     "test.me:80",
			expectedHostName: "test.me",
		},
		{
			serverAddr:       "https://test.me:80/",
			expectedScheme:   "https",
			expectedHost:     "test.me:80",
			expectedHostName: "test.me",
		},
		{
			serverAddr:       "https://test.me/",
			expectedScheme:   "https",
			expectedHost:     "test.me",
			expectedHostName: "test.me",
		},
		{
			serverAddr:       "test.me:8080",
			expectedScheme:   "https",
			expectedHost:     "test.me:8080",
			expectedHostName: "test.me",
		},
		{
			serverAddr:       "http://test.me:8080",
			expectedScheme:   "http",
			expectedHost:     "test.me:8080",
			expectedHostName: "test.me",
		},
	}

	for idx, tc := range testCases {
		gotScheme, gotHost, gotHostName, err := parseAKEApiService(tc.serverAddr)
		if err != nil {
			if !tc.expectedErr {
				t.Errorf("[%d] unexpected error: %v", idx, err)
			}
		} else {
			if tc.expectedErr {
				t.Errorf("[%d] expected error, but got nil", idx)
			} else {
				if gotScheme != tc.expectedScheme {
					t.Errorf("[%d] scheme not equal: expected %q, but got %q", idx, tc.expectedScheme, gotScheme)
				}
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
