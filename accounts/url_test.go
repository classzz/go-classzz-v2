// Copyright 2018 The go-classzz-v2 Authors
// This file is part of the go-classzz-v2 library.
//
// The go-classzz-v2 library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-classzz-v2 library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-classzz-v2 library. If not, see <http://www.gnu.org/licenses/>.

package accounts

import (
	"testing"
)

func TestURLParsing(t *testing.T) {
	url, err := parseURL("https://classzz.org")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if url.Scheme != "https" {
		t.Errorf("expected: %v, got: %v", "https", url.Scheme)
	}
	if url.Path != "classzz.org" {
		t.Errorf("expected: %v, got: %v", "classzz.org", url.Path)
	}

	_, err = parseURL("classzz.org")
	if err == nil {
		t.Error("expected err, got: nil")
	}
}

func TestURLString(t *testing.T) {
	url := URL{Scheme: "https", Path: "classzz.org"}
	if url.String() != "https://classzz.org" {
		t.Errorf("expected: %v, got: %v", "https://classzz.org", url.String())
	}

	url = URL{Scheme: "", Path: "classzz.org"}
	if url.String() != "classzz.org" {
		t.Errorf("expected: %v, got: %v", "classzz.org", url.String())
	}
}

func TestURLMarshalJSON(t *testing.T) {
	url := URL{Scheme: "https", Path: "classzz.org"}
	json, err := url.MarshalJSON()
	if err != nil {
		t.Errorf("unexpcted error: %v", err)
	}
	if string(json) != "\"https://classzz.org\"" {
		t.Errorf("expected: %v, got: %v", "\"https://classzz.org\"", string(json))
	}
}

func TestURLUnmarshalJSON(t *testing.T) {
	url := &URL{}
	err := url.UnmarshalJSON([]byte("\"https://classzz.org\""))
	if err != nil {
		t.Errorf("unexpcted error: %v", err)
	}
	if url.Scheme != "https" {
		t.Errorf("expected: %v, got: %v", "https", url.Scheme)
	}
	if url.Path != "classzz.org" {
		t.Errorf("expected: %v, got: %v", "https", url.Path)
	}
}

func TestURLComparison(t *testing.T) {
	tests := []struct {
		urlA   URL
		urlB   URL
		expect int
	}{
		{URL{"https", "classzz.org"}, URL{"https", "classzz.org"}, 0},
		{URL{"http", "classzz.org"}, URL{"https", "classzz.org"}, -1},
		{URL{"https", "classzz.org/a"}, URL{"https", "classzz.org"}, 1},
		{URL{"https", "abc.org"}, URL{"https", "classzz.org"}, -1},
	}

	for i, tt := range tests {
		result := tt.urlA.Cmp(tt.urlB)
		if result != tt.expect {
			t.Errorf("test %d: cmp mismatch: expected: %d, got: %d", i, tt.expect, result)
		}
	}
}
