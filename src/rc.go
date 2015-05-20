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

const (
	DriveResourceConfiguration = ".driverc"
)

type ResourceConfiguration struct {
	ExcludeOps string `json:"exclude-ops"`
}

func kvifyCommentedFile(p, comment string) (kvMap map[string]string, err error) {
	var clauses []string
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

func splitAndStrip(line string, resolveFromEnv bool) (kv keyValue, err error) {
	line = strings.Trim(line, " ")
	subStrCount := 2
	splits := strings.SplitN(line, "=", subStrCount)
	fmt.Println(splits)

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
		resolvedValue = os.Getenv(resolvedValue)
	}

	if resolvedValue != "" && joinedValue != "" {
		joinedValue = resolvedValue
	}

	kv.key = key
	kv.value = joinedValue

	return
}
