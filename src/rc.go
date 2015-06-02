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
	"os"
	"path"
	"regexp"
	"strings"
)

const (
	DriveResourceConfiguration = ".driverc"
)

var (
	HomeEnvKey           = "HOME"
	HomeShellEnvKey      = "$" + HomeEnvKey
	FsHomeDir            = os.Getenv(HomeEnvKey)
	ShellVarSplitsRegStr = "(\\$+[^\\s\\/]*)"
)

var (
	ShellVarRegCompile, _ = regexp.Compile(ShellVarSplitsRegStr)
)

func envResolved(v string) string {

	matches := ShellVarRegCompile.FindAll([]byte(v), -1)
	if len(matches) < 1 {
		return v
	}

	uniq := make(map[string]string)

	for _, match := range matches {
		strMatch := string(match)
		if _, ok := uniq[strMatch]; ok {
			continue
		}
		uniq[strMatch] = strMatch
	}

	for envKey, _ := range uniq {
		qKey := strings.Replace(envKey, "$", "", -1)
		resolved := os.Getenv(qKey)
		v = strings.Replace(v, envKey, resolved, -1)
	}

	return v
}

func kvifyCommentedFile(p, comment string) (kvMap map[string]string, err error) {
	var clauses []string
	kvMap = make(map[string]string)
	clauses, err = readCommentedFile(p, comment)
	if err != nil {
		return
	}

	for i, clause := range clauses {
		kvf, kvErr := splitAndStrip(clause, true)
		if kvErr != nil {
			err = kvErr
			return
		}

		value, ok := kvf.value.(string)
		if !ok {
			err = fmt.Errorf("clause %d: expected a string instead got %v", i, kvf.value)
			return
		}

		kvMap[kvf.key] = value
	}
	return
}

const (
	TBool = iota
	TString
	TStringArray
	TByteArray
	TInt
	TInt64
)

type typeEmitterPair struct {
	value        string
	valueEmitter func(v string) interface{}
}

func rcMapToOptions(rcMap map[string]string) (*Options, error) {
	targetKeys := []string{
		CoercedMimeKeyKey,
		ForceKey,
		QuietKey,
		HiddenKey,
		NoPromptKey,
	}

	accepted := make(map[string]string)
	for _, key := range targetKeys {
		retr, ok := rcMap[key]
		if !ok {
			continue
		}

		accepted[key] = retr
	}

	if len(accepted) < 1 {
		return nil, fmt.Errorf("rcMapping: no keys matched")
	}

	return nil, fmt.Errorf("not yet implemented")
}

func rcPath(absDirPath string) string {
	return path.Join(absDirPath, DriveResourceConfiguration)
}

func globalRcFile() string {
	ps := rcPath(FsHomeDir)
	return ps
}

func splitAndStrip(line string, resolveFromEnv bool) (kv keyValue, err error) {
	line = strings.Trim(line, " ")
	subStrCount := 2
	splits := strings.SplitN(line, "=", subStrCount)

	splitsLen := len(splits)
	if splitsLen < subStrCount-1 {
		return
	}

	if splitsLen < subStrCount {
		err = fmt.Errorf("expected <key>=<value> instead got %v", splits[:splitsLen])
		return
	}

	key, value := splits[0], splits[1:]
	joinedValue := strings.Join(NonEmptyTrimmedStrings(value...), " ")

	joinedValue = strings.Trim(joinedValue, " ")
	resolvedValue := joinedValue

	if resolveFromEnv {
		resolvedValue = envResolved(resolvedValue)
	}

	if resolvedValue != "" && joinedValue != "" {
		joinedValue = resolvedValue
	}

	kv.key = key
	kv.value = joinedValue

	return
}
