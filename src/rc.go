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

func rcFileToOptions(rcPath string) (*Options, error) {
	rcMap, err := kvifyCommentedFile(rcPath, CommentStr)
	if err != nil {
		return nil, err
	}
	return rcMapToOptions(rcMap)
}

func ResourceConfigurationToOptions(path string) (*Options, error) {
	beginOpts := Options{Path: path}
	rcP, rcErr := beginOpts.rcPath()
	if rcErr != nil {
		return nil, rcErr
	}

	return rcFileToOptions(rcP)
}

func rcMapToOptions(rcMap map[string]string) (*Options, error) {
	targetKeys := []typeResolver{
		{
			key: ForceKey, resolver: boolfer,
		},
		{
			key: QuietKey, resolver: boolfer,
		},
		{
			key: HiddenKey, resolver: boolfer,
		},
		{
			key: NoPromptKey, resolver: boolfer,
		},
		{
			key: NoClobberKey, resolver: boolfer,
		},
		{
			key: IgnoreConflictKey, resolver: boolfer,
		},
		{
			key: RecursiveKey, resolver: boolfer,
		},

		{
			key: DepthKey, resolver: intfer,
		},
		{
			key: ExportsDirKey, resolver: stringfer,
		},
		{
			key: ExportsKey, resolver: stringArrayfer,
		},
		{
			key: ExcludeOpsKey, resolver: stringArrayfer,
		},
	}

	accepted := make(map[string]typeResolver)
	for _, stK := range targetKeys {
		lowerKey := strings.ToLower(stK.key)
		retr, ok := rcMap[lowerKey]
		if !ok {
			continue
		}

		stK.value = retr
		accepted[lowerKey] = stK
	}

	if len(accepted) < 1 {
		return nil, fmt.Errorf("rcMapping: no keys matched")
	}

	if false {
		fmt.Println("rcMap", rcMap, "accepted", accepted)
	}

	opts := &Options{}
	for lowerKey, stK := range accepted {
		if err := stK.resolver(lowerKey, stK.value, opts); err != nil {
			return nil, err
		}
	}

	return opts, nil
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
