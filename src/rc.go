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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/odeke-em/namespace"
)

var (
	ErrRcNoKeysMatched = errors.New("rcMapping: no keys matched")
)

const (
	DriveResourceConfiguration = ".driverc"
)

var (
	HomeEnvKey      = "HOME"
	HomeShellEnvKey = "$" + HomeEnvKey
	FsHomeDir       = os.Getenv(HomeEnvKey)
)

func kvifyCommentedFile(p, comment string) (map[string]map[string]string, error) {
	clauses, err := readCommentedFile(p, comment)
	if err != nil {
		return nil, err
	}

	linesChan := make(chan string)
	go func() {
		defer close(linesChan)
		for _, clause := range clauses {
			linesChan <- clause
		}
	}()

	ns, err := namespace.ParseCh(linesChan)
	if err != nil {
		return nil, err
	}

	gkvMap := make(map[string]map[string]string)
	for key, clauses := range ns {
		kvMap := make(map[string]string)
		for i, clause := range clauses {
			kvf, kvErr := splitAndStrip(clause, true)
			if kvErr != nil {
				return nil, kvErr
			}

			value, ok := kvf.value.(string)
			if !ok {
				err = fmt.Errorf("clause %d: expected a string instead got %v", i, kvf.value)
				return nil, err
			}

			kvMap[kvf.key] = value
		}
		gkvMap[key] = kvMap
	}
	return gkvMap, nil
}

func ResourceMappings(rcPath string) (map[string]map[string]interface{}, error) {
	beginOpts := Options{Path: rcPath}
	rcPath, rcErr := beginOpts.rcPath()

	if rcErr != nil {
		DebugPrintf("tried to read from rcPath: %s got err: %v", rcPath, rcErr)
		return nil, rcErr
	}

	DebugPrintf("RCPath: %s", rcPath)
	nsRCMap, rErr := kvifyCommentedFile(rcPath, CommentStr)
	if rErr != nil {
		return nil, rErr
	}

	grouped := make(map[string]map[string]interface{})
	for key, ns := range nsRCMap {
		parsed, err := parseRCValues(ns)
		if err != nil {
			return nil, err
		}
		grouped[key] = parsed
	}

	if jsonRepr, err := json.MarshalIndent(grouped, "", "  "); err == nil {
		DebugPrintf("parsedContent from %q\n%s", rcPath, jsonRepr)
	} else {
		DebugPrintf("parsedContent from %q\n%s", rcPath, grouped)
	}

	return grouped, nil
}

func parseRCValues(rcMap map[string]string) (valueMappings map[string]interface{}, err error) {
	valueMappings = make(map[string]interface{})

	targetKeys := []struct {
		resolver resolverEmitter
		keys     []string
	}{
		{
			resolver: _boolfer, keys: []string{
				OcrKey, ConvertKey, CLIOptionFileBrowser, CLIOptionWebBrowser,
				CLIOptionVerboseKey, RecursiveKey, CLIOptionFiles, CLIOptionLongFmt,
				ForceKey, QuietKey, HiddenKey, NoPromptKey, NoClobberKey, IgnoreConflictKey,
				CLIOptionIgnoreNameClashes, CLIOptionIgnoreChecksum, CLIOptionFixClashesKey,
				CLIOptionDesktopLinks, CLIOptionExportsDumpToSameDirectory, CLIOptionTrashed,
				CLIOptionStarred, CLIOptionPiped, CLIOptionExplicitlyExport,
				CLIOptionDirectories, CLIOptionAllStarred,
			},
		},
		{
			resolver: _intfer, keys: []string{
				PageSizeKey,
				DepthKey,
				CLIOptionRetryCount,
			},
		},
		{
			resolver: _stringfer, keys: []string{
				CLIOptionUnified, CLIOptionDiffBaseLocal,
				ExportsKey, ExcludeOpsKey, CLIOptionUnifiedShortKey,
				CLIEncryptionPassword, CLIDecryptionPassword, SortKey,
				CLIOptionNotOwner, ExportsDirKey, CLIOptionExactTitle, AddressKey,
				CLIOptionPushDestination, CLIOptionSkipMime, CLIOptionMatchMime,
				ExportsKey,
			},
		},
		{
			resolver: _stringArrayfer, keys: []string{
			// Add items that might need string array parsing and conversion here
			},
		},
	}

	accepted := make(map[string]typeResolver)
	for _, item := range targetKeys {
		resolver := item.resolver
		keys := item.keys
		for _, key := range keys {
			lowerKey := strings.ToLower(key)
			retr, ok := rcMap[lowerKey]
			if !ok {
				continue
			}

			tr := typeResolver{key: key, value: retr, resolver: resolver}
			accepted[lowerKey] = tr
		}
	}

	if false && len(accepted) < 1 {
		err = ErrRcNoKeysMatched
		return
	}

	for lowerKey, tResolver := range accepted {
		if value, err := tResolver.resolver(lowerKey, tResolver.value); err == nil {
			valueMappings[lowerKey] = value
		} else {
			fmt.Fprintf(os.Stderr, "rc: %s err %v\n", lowerKey, err)
		}
	}

	return
}

func rcMapToOptions(rcMap map[string]string) (*Options, error) {
	opts := &Options{}

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
		resolvedValue = os.ExpandEnv(resolvedValue)
	}

	if resolvedValue != "" && joinedValue != "" {
		joinedValue = resolvedValue
	}

	kv.key = key
	kv.value = joinedValue

	return
}

func JSONStringifySiftedCLITags(from interface{}, rcSourcePath string, defined map[string]bool, relevantNamespaces ...string) (string, error) {
	rcMappings, err := ResourceMappings(rcSourcePath)
	if err != nil && !NotExist(err) {
		return "", err
	}

	cs := CliSifter{
		From:           from,
		Defaults:       mergeNamespaces(rcMappings, relevantNamespaces...),
		AlreadyDefined: defined,
	}

	return SiftCliTags(&cs), nil
}

func copyAndOverWriteNs(from, to map[string]interface{}) {
	for fK, fV := range from {
		to[fK] = fV
	}
}

func mergeNamespaces(ns map[string]map[string]interface{}, relevantKeys ...string) map[string]interface{} {
	// Start with the global namespace and proceed overwriting from specific keys
	var combinedNs map[string]interface{} = ns[namespace.GlobalNamespaceKey]
	if combinedNs == nil {
		combinedNs = make(map[string]interface{})
	}

	for _, key := range relevantKeys {
		kNs := ns[key]
		if len(kNs) < 1 {
			// TODO: Decide if not setting anything in the namespace
			// means exclude it entirely hence make combinedNs nil?
			continue
		}

		// Now merge them: from -> to
		copyAndOverWriteNs(kNs, combinedNs)
	}

	return combinedNs
}
