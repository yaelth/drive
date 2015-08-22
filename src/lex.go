// Copyright 2015 Google Inc. All Rights Reserved.
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
	"strconv"
	"strings"
)

const (
	TBool = iota
	TRune
	TInt
	TInt64
	TUInt
	TString
	TStringArray
	TUInt64
)

type typeEmitterPair struct {
	value        string
	valueEmitter func(v string) interface{}
}

type typeResolver struct {
	key      string
	value    string
	resolver func(varname, v string, optSave *Options) error
}

type tParsed int

func (t tParsed) String() string {
	switch t {
	case TBool:
		return "bool"
	case TInt:
		return "int"
	case TInt64:
		return "int64"
	case TRune:
		return "rune"
	case TString:
		return "string"
	case TStringArray:
		return "[]string"
	case TUInt64:
		return "uint64"
	default:
		return "unknown"
	}
}

func parseErrorer(varname string, t tParsed, value interface{}, args ...interface{}) error {
	return fmt.Errorf("%s: got \"%v\", expected type %s %v", varname, value, t, args)
}

func boolfer(varname string, value string, optSave *Options) error {
	retr, err := strconv.ParseBool(value)
	if err != nil {
		return parseErrorer(varname, TBool, value, err)
	}

	var ptr *bool
	switch strings.ToLower(varname) {
	case QuietKey:
		ptr = &optSave.Quiet
	case HiddenKey:
		ptr = &optSave.Hidden
	case NoClobberKey:
		ptr = &optSave.NoClobber
	case ForceKey:
		ptr = &optSave.Force
	case IgnoreChecksumKey:
		ptr = &optSave.IgnoreChecksum
	case NoPromptKey:
		ptr = &optSave.NoPrompt
	case IgnoreConflictKey:
		ptr = &optSave.IgnoreConflict
	case RecursiveKey:
		ptr = &optSave.Recursive
	case IgnoreNameClashesKey:
		ptr = &optSave.IgnoreNameClashes
	}

	if ptr != nil {
		*ptr = retr
	}

	return nil
}

func stringfer(varname, value string, optSave *Options) error {
	// TODO: perform strips and trims
	retr := value

	var ptr *string
	switch strings.ToLower(varname) {
	case ExportsDirKey:
		ptr = &optSave.ExportsDir
	}

	if ptr != nil {
		*ptr = retr
	}

	return nil
}

func stringArrayfer(varname, value string, optSave *Options) error {
	splits := NonEmptyTrimmedStrings(strings.Split(value, ",")...)
	var ptr *[]string

	varnameLower := strings.ToLower(varname)

	switch varnameLower {
	case ExportsKey:
		ptr = &optSave.Exports

	case ExcludeOpsKey:
		optSave.ExcludeCrudMask = CrudAtoi(splits...)
		return nil
	}

	if ptr != nil {
		*ptr = splits
	}
	return nil
}

func int64fer(varname, value string, optSave *Options) error {
	retr, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return parseErrorer(varname, TInt64, value, err)
	}

	var ptr *int64
	switch strings.ToLower(varname) {
	case PageSizeKey:
		ptr = &optSave.PageSize
	}

	if ptr != nil {
		*ptr = retr
	}
	return nil
}

func intfer(varname, value string, optSave *Options) error {
	retr64, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return parseErrorer(varname, TInt, value, err)
	}

	var ptr *int
	switch strings.ToLower(varname) {
	case DepthKey:
		ptr = &optSave.Depth
	}

	if ptr != nil {
		*ptr = int(retr64)
	}
	return nil
}

func CopyOptionsFromKeysIfNotSet(fromPtr, toPtr *Options, alreadySetKeys map[string]bool) {
	from := *fromPtr
	fromValue := reflect.ValueOf(from)
	toValue := reflect.ValueOf(toPtr).Elem()

	fromType := reflect.TypeOf(from)

	for i, n := 0, fromType.NumField(); i < n; i++ {
		fromFieldT := fromType.Field(i)
		fromTag := fromFieldT.Tag.Get("cli")

		_, alreadySet := alreadySetKeys[fromTag]
		if alreadySet {
			continue
		}

		fromFieldV := fromValue.Field(i)
		toFieldV := toValue.Field(i)

		if !toFieldV.CanSet() {
			continue
		}

		toFieldV.Set(fromFieldV)
	}
}
