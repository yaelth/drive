// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package drive

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"google.golang.org/api/googleapi"
)

func TestRemoteOpToChangerTranslator(t *testing.T) {
	g := &Commands{}
	now := time.Now()

	cases := []struct {
		change   *Change
		name     string
		wantedFn func(*Change) error
	}{
		{change: &Change{Src: nil, Dest: nil}, wantedFn: nil, name: "nil"},
		{change: &Change{Src: &File{}, Dest: &File{}}, wantedFn: nil, name: "nil"},
		{change: &Change{Src: &File{}, Dest: nil}, wantedFn: g.remoteAdd, name: "remoteAdd"},
		{change: &Change{Src: nil, Dest: &File{}}, wantedFn: g.remoteTrash, name: "remoteTrash"},
		{
			change: &Change{
				Dest: &File{ModTime: now},
				Src:  &File{ModTime: now.Add(time.Hour)},
			},
			wantedFn: g.remoteMod, name: "remoteMod",
		},
		{
			change: &Change{
				Dest: &File{ModTime: now},
				Src:  &File{ModTime: now},
			},
			wantedFn: nil, name: "noop",
		},
	}

	for _, tc := range cases {
		got := remoteOpToChangerTranslator(g, tc.change)
		vptr1 := reflect.ValueOf(got).Pointer()
		vptr2 := reflect.ValueOf(tc.wantedFn).Pointer()

		if vptr1 != vptr2 {
			t.Errorf("expected %q expected (%v) got (%v)", tc.name, tc.wantedFn, got)
		}
	}
}

func TestLocalOpToChangerTranslator(t *testing.T) {
	g := &Commands{}
	now := time.Now()

	cases := []struct {
		change   *Change
		name     string
		wantedFn func(*Change, []string) error
	}{
		{change: &Change{Src: nil, Dest: nil}, wantedFn: nil, name: "nil"},
		{change: &Change{Src: &File{}, Dest: &File{}}, wantedFn: nil, name: "nil"},
		{
			change:   &Change{Src: &File{}, Dest: nil},
			wantedFn: g.localAdd, name: "localAdd",
		},
		{
			change:   &Change{Dest: nil, Src: &File{}},
			wantedFn: g.localAdd, name: "localAdd",
		},
		{
			change:   &Change{Src: nil, Dest: &File{}},
			wantedFn: g.localDelete, name: "localDelete",
		},
		{
			change: &Change{
				Src:  &File{ModTime: now},
				Dest: &File{ModTime: now.Add(time.Hour)},
			},
			wantedFn: g.localMod, name: "localMod",
		},
		{
			change: &Change{
				Dest: &File{ModTime: now},
				Src:  &File{ModTime: now},
			},
			wantedFn: nil, name: "noop",
		},
	}

	for _, tc := range cases {
		got := localOpToChangerTranslator(g, tc.change)
		vptr1 := reflect.ValueOf(got).Pointer()
		vptr2 := reflect.ValueOf(tc.wantedFn).Pointer()

		if vptr1 != vptr2 {
			t.Errorf("expected %q expected (%v) got (%v)", tc.name, tc.wantedFn, got)
		}
	}
}

func TestRetryableErrorCheck(t *testing.T) {
	cases := []struct {
		value              interface{}
		success, retryable bool
		comment            string
	}{
		{
			value: nil, success: false, retryable: true,
			comment: "a nil tuple is retryable but not successful",
		},
		{
			value: t, success: false, retryable: true,
			comment: "t value is not a tuple, is retryable but not successful",
		},
		{
			value: &tuple{first: nil, last: nil}, success: true, retryable: false,
			comment: "last=nil representing a nil error so success, unretryable",
		},
		{
			value:   &tuple{first: nil, last: fmt.Errorf("flux")},
			success: false, retryable: true,
			comment: "last!=nil, non-familiar error so unsuccessful, retryable",
		},
		{
			value: &tuple{
				first: "",
				last: &googleapi.Error{
					Message: "This is an error",
				},
			},
			success: false, retryable: false,
			comment: "last!=nil, familiar error so unsuccessful, retryable:: statusCode undefined",
		},
		{
			value: &tuple{
				first: "",
				last: &googleapi.Error{
					Code: 500,
					Message: "This is an error",
				},
			},
			success: false, retryable: true,
			comment: "last!=nil, familiar error so unsuccessful, retryable:: statusCode 500",
		},
		{
			value: &tuple{
				first: nil, 
				last: &googleapi.Error{
					Code: 401,
					Message: "401 right here",
				},
			},
			success: false, retryable: true,
			comment: "last!=nil, 401 must be retryable",
		},
		{
			value: &tuple{
				first: nil, 
				last: &googleapi.Error{
					Code: 409,
					Message: "409 right here",
				},
			},
			success: false, retryable: false,
			comment: "last!=nil, 409 is unclassified so unretryable",
		},
		{
			value: &tuple{
				first: nil, 
				last: &googleapi.Error{
					Code: 403,
					Message: "403 right here",
				},
			},
			success: false, retryable: true,
			comment: "last!=nil, 403 is retryable",
		},
		{
			value: &tuple{
				first: nil, 
				last: &googleapi.Error{
					Code: 500,
					Message: MsgErrFileNotMutable,
				},
			},
			success: false, retryable: false,
			comment: "issue #472 FileNotMutable is unretryable",
		},
		{
			value: &tuple{
				first: nil, 
				last: &googleapi.Error{
					Code: 500,
					Message: strings.ToLower(MsgErrFileNotMutable),
				},
			},
			success: false, retryable: false,
			comment: "issue #472 FileNotMutable is unretryable, casefold held",
		},
		{
			value: &tuple{
				first: nil, 
				last: &googleapi.Error{
					Code: 501,
					Message: strings.ToUpper(MsgErrFileNotMutable),
				},
			},
			success: false, retryable: false,
			comment: "issue #472 FileNotMutable is unretryable, casefold held",
		},
	}

	for _, tc := range cases {
		success, retryable := retryableErrorCheck(tc.value)
		if success != tc.success {
			t.Errorf("%v success got %v expected %v", tc.value, success, tc.success)
		}
		if retryable != tc.retryable {
			t.Errorf("%v retryable got %v expected %v: %q", tc.value, retryable, tc.retryable, tc.comment)
		}
	}
}
