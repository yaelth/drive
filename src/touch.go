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
	"time"
)

func multiplexOnChanMapResults(g *Commands, chanMap map[int]chan *keyValue) {
	spin := g.playabler()
	spin.play()
	defer spin.stop()

	for {
		if len(chanMap) < 1 {
			break
		}
		// Find the channel that has results
		for key, kvChan := range chanMap {
			select {
			case kv := <-kvChan:
				if kv == nil { // Sentinel emitted
					delete(chanMap, key)
					continue
				}
				if kv.value != nil {
					g.log.LogErrf("touch: %s %v\n", kv.key, kv.value.(error))
				}
			default:
			}
		}
	}

	return
}

func (g *Commands) Touch(byId bool) (err error) {
	// Arbitrary value for rate limiter
	throttle := time.Tick(1e9 / 10)

	chanMap := map[int]chan *keyValue{}
	touchModTime, err := g.requestedTouchModTime()
	if err != nil {
		return err
	}

	for i, relToRootPath := range g.opts.Sources {
		fileId := ""
		if byId {
			fileId = relToRootPath
		}
		chanMap[i] = g.touch(relToRootPath, fileId, g.opts.Depth, touchModTime)
		<-throttle
	}

	multiplexOnChanMapResults(g, chanMap)
	return
}

const (
	DefaultTouchTimeSpecifier = "20060102150405"
)

func (g *Commands) requestedTouchModTime() (*time.Time, error) {
	var requestedModTime *time.Time
	metaPtr := g.opts.Meta
	if metaPtr == nil {
		return requestedModTime, nil
	}

	meta := *metaPtr

	formatSpecifiers := meta[TouchTimeFmtSpecifierKey]
	formatSpecifiers = append(formatSpecifiers, DefaultTouchTimeSpecifier)
	modDateStrL, ok := meta[TouchModTimeKey]
	if ok && len(modDateStrL) >= 1 {
		return parseDate(modDateStrL[0], formatSpecifiers...)
	}

	// Otherwise for last resort try parsing by duration
	durationOffsetStrL, ok := meta[TouchOffsetDurationKey]
	if ok && len(durationOffsetStrL) >= 1 {
		return parseDurationOffsetFromNow(durationOffsetStrL[0])
	}

	return requestedModTime, nil
}

func (g *Commands) TouchByMatch() (err error) {
	mq := matchQuery{
		dirPath: g.opts.Path,
		inTrash: false,
		titleSearches: []fuzzyStringsValuePair{
			{fuzzyLevel: Like, values: g.opts.Sources, inTrash: false},
		},
	}

	touchModTime, err := g.requestedTouchModTime()
	if err != nil {
		return err
	}

	pagePair := g.rem.FindMatches(&mq)
	errsChan := pagePair.errsChan
	matchesChan := pagePair.filesChan

	throttle := time.Tick(1e9 / 10)
	chanMap := map[int]chan *keyValue{}

	i := 0
	working := true
	for working {
		select {
		case err := <-errsChan:
			if err != nil {
				return err
			}
		case match, stillHasContent := <-matchesChan:
			if !stillHasContent {
				working = false
				break
			}
			if match == nil {
				continue
			}

			chanMap[i] = g.touch(g.opts.Path+"/"+match.Name, match.Id, g.opts.Depth, touchModTime)
			<-throttle
			i += 1
		}
	}

	multiplexOnChanMapResults(g, chanMap)
	return
}

// resolveTouch figures out which function to invoke
// in order to perform a touch/modTime change of a file.
func (g *Commands) resolveTouch(fileId, relToRootPath string, modTime *time.Time) (*File, error) {
	if fileId == "" {
		return g.touchByPath(relToRootPath, modTime)
	}

	// Now dealing with only fileId
	if modTime == nil {
		return g.rem.Touch(fileId)
	}

	return g.rem.SetModTime(fileId, *modTime)
}

func (g *Commands) touch(relToRootPath, fileId string, depth int, modTime *time.Time) chan *keyValue {
	fileChan := make(chan *keyValue)

	go func() {
		kv := &keyValue{
			key:   relToRootPath,
			value: nil,
		}

		defer func() {
			fileChan <- kv
			close(fileChan)
		}()

		file, err := g.resolveTouch(fileId, relToRootPath, modTime)

		if err != nil {
			kv.value = err
			return
		}

		if g.opts.Verbose {
			g.log.Logf("%s: %v\n", relToRootPath, file.ModTime)
		}

		depth = decrementTraversalDepth(depth)
		if depth == 0 {
			return
		}

		if file.IsDir {
			childResults := make(chan chan *keyValue)
			// Arbitrary value for rate limiter
			throttle := time.Tick(1e9 * 2)
			childrenPagePair := g.rem.FindByParentId(file.Id, g.opts.Hidden)
			errsChan := childrenPagePair.errsChan
			childrenChan := childrenPagePair.filesChan

			go func() {
				defer close(childResults)

				working := true
				for working {
					select {
					case err := <-errsChan:
						g.log.LogErrf("%v", err)
					case child, stillHasContent := <-childrenChan:
						if !stillHasContent {
							working = false
							break
						}
						if child == nil {
							working = false
							break
						}

						childResults <- g.touch(relToRootPath+"/"+child.Name, child.Id, depth, modTime)
						<-throttle
					}
				}
			}()

			for childChan := range childResults {
				for childFile := range childChan {
					fileChan <- childFile
				}
			}
		}
	}()

	return fileChan
}

func (g *Commands) touchByPath(relToRootPath string, modTime *time.Time) (*File, error) {
	file, err := g.rem.FindByPath(relToRootPath)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, ErrPathNotExists
	}
	if modTime == nil {
		return g.rem.Touch(file.Id)
	}

	return g.rem.SetModTime(file.Id, *modTime)
}
