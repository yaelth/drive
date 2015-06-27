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
	"strconv"
	"sync"
	"time"

	"github.com/odeke-em/drive/config"
)

func (g *Commands) Fetch() (err error) {
	deletions, delErr := g.pruneMissingIndices()
	if delErr != nil {
		return delErr
	}

	for del := range deletions {
		if del == nil {
			continue
		}
		delErr = os.Remove(del.BlobAt)

		if delErr != nil {
			g.log.LogErrf("fetch removing index for: \"%s\" at \"%s\" %v\n", del.Name, del.BlobAt, delErr)
		}
	}

	return nil
}

func (g *Commands) FetchById() (err error) {
	return g.fetch(true)
}

func (g *Commands) fetch(byId bool) (err error) {
	cl, _, err := pullLikeResolve(g, byId)

	if err != nil {
		return err
	}

	ok, opMap := printChangeList(g.log, cl, !g.opts.canPrompt(), g.opts.NoClobber)
	if !ok {
		return nil
	}

	return g.playFetchChanges(cl, opMap)
}

func loneCountRegister(wg *sync.WaitGroup, progress chan int) {
	progress <- 1
	wg.Done()
	return
}

func (g *Commands) playFetchChanges(cl []*Change, opMap *map[Operation]sizeCounter) (err error) {
	changeCount := len(cl)

	g.taskStart(int64(changeCount))

	defer close(g.rem.progressChan)

	// sort.Sort(ByPrecedence(cl))

	go func() {
		for n := range g.rem.progressChan {
			g.taskAdd(int64(n))
		}
	}()

	var wg sync.WaitGroup
	wg.Add(changeCount)

	ticker := time.Tick(1e9 / 10)

	for _, c := range cl {
		switch c.Op() {
		case OpDelete:
			go g.removeIndex(&wg, c.Dest)
		case OpNone:
			loneCountRegister(&wg, g.rem.progressChan)
		default:
			go g.addIndex(&wg, c.Src)
		}

		<-ticker
	}

	wg.Wait()

	return nil
}

func (g *Commands) addIndex(wg *sync.WaitGroup, f *File) (err error) {
	defer loneCountRegister(wg, g.rem.progressChan)

	index := f.ToIndex()
	wErr := g.context.SerializeIndex(index, g.context.AbsPathOf(""))

	// TODO: Should indexing errors be reported?
	if wErr != nil {
		g.log.LogErrf("addIndex %s: %v\n", f.Name, wErr)
	}

	return wErr
}

func (g *Commands) removeIndex(wg *sync.WaitGroup, f *File) (err error) {
	defer loneCountRegister(wg, g.rem.progressChan)
	if f.Id == "" {
		return
	}

	index := f.ToIndex()
	rmErr := g.context.RemoveIndex(index, g.context.AbsPathOf(""))

	// TODO: Should indexing errors be reported?
	if rmErr != nil {
		g.log.LogErrf("removeIndex %s: %v\n", f.Name, rmErr)
	}

	return rmErr
}

func queryIdify(ids map[string]*File) string {
	idified := []string{}

	for id, _ := range ids {
		idified = append(idified, fmt.Sprintf("(id=%s)", strconv.Quote(id)))
	}

	return sepJoin(" or ", idified...)
}

func mapifyFiles(g *Commands, ids map[string]*File) (map[string]*File, error) {
	delivery := make(chan *File)
	mapping := make(map[string]*File)

	for id, _ := range ids {
		go func(qId string) {
			f, err := g.rem.FindById(qId)
			if false && err != nil && err != ErrPathNotExists {
				g.log.LogErrln(id, err, "x")
			}
			delivery <- f
		}(id)
	}

	doneCount := len(ids)
	for i := 0; i < doneCount; i += 1 {
		f := <-delivery
		if f == nil {
			continue
		}

		mapping[f.Id] = f
	}

	return mapping, nil
}

func (g *Commands) pruneMissingIndices() (deletions chan *File, err error) {
	var listing chan *File
	indicesDir := config.IndicesAbsPath("", "")
	listing, err = list(g.context, indicesDir, true, nil)

	deletions = make(chan *File)

	if err != nil {
		close(deletions)
		return
	}

	go func() {

		spin := g.playabler()

		defer func() {
			spin.stop()
			close(deletions)
		}()

		queriesPerRequest := 10
		tick := time.Tick(time.Duration(1e9 / queriesPerRequest))

		iterating := true

		doneCount := uint64(0)

		for iterating {
			spin.play()

			localIds := map[string]*File{}
			i := 0
			for i < queriesPerRequest {
				i += 1
				f, ok := <-listing
				if !ok {
					iterating = false
					break
				}

				localIds[f.Name] = f
			}

			if len(localIds) < 1 {
				break
			}

			mapping, mErr := mapifyFiles(g, localIds)
			if mErr != nil {
				g.log.LogErrf("syncIndices: %v\n", mErr)
				continue
			}

			delCount := 0
			for localId, localFile := range localIds {
				_, ok := mapping[localId]
				if !ok {
					delCount += 1
					deletions <- localFile
				}
			}

			doneCount += uint64(i)

			spin.pause()
			deletionsReport := ""
			if delCount >= 1 {
				deletionsReport = fmt.Sprintf("%v/%v to be deleted", delCount, i)
			}

			g.log.LogErrf("\rfetch: %s %v processed so far\r", deletionsReport, doneCount)
			<-tick
		}
	}()

	return
}
