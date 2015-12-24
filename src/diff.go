// Copyright 2013 Google Inc. All Rights Reserved.
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
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"
)

// MaxFileSize is the max number of bytes we
// can accept for diffing (Arbitrary value)
const MaxFileSize = 50 * 1024 * 1024

var Ruler = strings.Repeat("*", 4)

const (
	DiffNone = 1 << iota
	DiffUnified
)

type diffSt struct {
	change       *Change
	diffProgPath string
	cwd          string
	mask         int
	printRuler   bool
	// baseLocal when set uses local as the base otherwise remote is used as the base
	baseLocal bool
}

func (d diffSt) unified() bool {
	return (d.mask & DiffUnified) != 0
}

func (g *Commands) Diff() (err error) {
	var cl []*Change

	spin := g.playabler()
	spin.play()
	defer spin.stop()

	clashes := []*Change{}
	for _, relToRootPath := range g.opts.Sources {
		fsPath := g.context.AbsPathOf(relToRootPath)
		ccl, cclashes, cErr := g.changeListResolve(relToRootPath, fsPath, true)
		// TODO: Show the conflicts if any

		clashes = append(clashes, cclashes...)
		if cErr != nil {
			if cErr != ErrClashesDetected {
				return cErr
			} else {
				err = reComposeError(err, cErr.Error())
			}
		}

		if len(ccl) > 0 {
			cl = append(cl, ccl...)
		}
	}

	if !g.opts.IgnoreNameClashes && len(clashes) >= 1 {
		warnClashesPersist(g.log, clashes)
		if err != nil {
			return err
		}
		return ErrClashesDetected
	}

	spin.stop()

	var diffUtilPath string
	diffUtilPath, err = exec.LookPath("diff")
	if err != nil {
		return
	}

	dst := diffSt{
		diffProgPath: diffUtilPath,
		cwd:          ".",
		mask:         g.opts.TypeMask,
		printRuler:   len(cl) > 1,
		baseLocal:    g.opts.BaseLocal,
	}

	for _, c := range cl {
		dst.change = c
		dErr := g.perDiff(dst)
		if dErr != nil {
			g.log.LogErrln(dErr)
		}
	}
	return
}

func (g *Commands) perDiff(dSt diffSt) (err error) {
	change := dSt.change
	diffProgPath, cwd := dSt.diffProgPath, dSt.cwd

	l, r := change.Src, change.Dest
	if l == nil && r == nil {
		return fmt.Errorf("Neither remote nor local exists")
	}
	if r == nil && l != nil {
		return fmt.Errorf("%s only on local", change.Path)
	}
	if l == nil && r != nil {
		return fmt.Errorf("%s only on remote", change.Path)
	}
	// Pre-screening phase
	if r.IsDir && l.IsDir {
		return fmt.Errorf("Both local and remote are directories")
	}
	if r.IsDir && !l.IsDir {
		return fmt.Errorf("Remote is a directory while local is an ordinary file")
	}

	if l.IsDir && !r.IsDir {
		return fmt.Errorf("Local is a directory while remote is an ordinary file")
	}

	if r.BlobAt == "" {
		return fmt.Errorf("Cannot access download link for '%v'", r.Name)
	}

	if r.Size > MaxFileSize {
		return fmt.Errorf("%s Remote too large for display \033[94m[%v bytes]\033[00m",
			change.Path, r.Size)
	}
	if l.Size > MaxFileSize {
		return fmt.Errorf("%s Local too large for display \033[92m[%v bytes]\033[00m",
			change.Path, l.Size)
	}

	mask := fileDifferences(r, l, g.opts.IgnoreChecksum)
	if mask == DifferNone {
		// No output when "no changes found"
		return nil
	}

	typeName := "File"
	if l.IsDir {
		typeName = "Directory"
	}
	g.log.Logf("%s: %s\n", typeName, change.Path)

	if modTimeDiffers(mask) {
		g.log.Logf("* %-15s %-40s\n* %-15s %-40s\n",
			"local:", toUTCString(l.ModTime), "remote:", toUTCString(r.ModTime))

		if mask == DifferModTime { // No further change
			return
		}
	}

	defer func() {
		if dSt.printRuler {
			g.log.Logf("\n%s\n", Ruler)
		}
	}()

	var frTmp, fl *os.File
	var blob io.ReadCloser

	// Clean-up
	defer func() {
		if frTmp != nil {
			os.RemoveAll(frTmp.Name())
		}
		if fl != nil {
			fl.Close()
		}
		if blob != nil {
			blob.Close()
		}
	}()

	blob, err = g.rem.Download(r.Id, "")
	if err != nil {
		return err
	}

	// Next step: Create a temp file with an obscure name unlikely to clash.
	tmpName := strings.Join([]string{
		".",
		fmt.Sprintf("tmp%v.tmp", rand.Int()),
	}, "x")

	frTmp, err = ioutil.TempFile(".", tmpName)
	if err != nil {
		return
	}
	_, err = io.Copy(frTmp, blob)
	if err != nil {
		return
	}

	if l.Name != r.Name {
		g.log.Logf("%s %s\n%s\n\n", l.Name, r.Name, Ruler)
	} else {
		g.log.Logf("%s\n%s\n\n", l.Name, Ruler)
	}

	diffArgs := []string{diffProgPath}
	if dSt.unified() {
		diffArgs = append(diffArgs, "-u")
	}

	// Next step: Determine which is the base file between local and remote
	first, other := frTmp.Name(), l.BlobAt
	if dSt.baseLocal {
		first, other = l.BlobAt, frTmp.Name()
	}

	diffArgs = append(diffArgs, first, other)
	diffCmd := exec.Cmd{
		Args:   diffArgs,
		Dir:    cwd,
		Path:   diffProgPath,
		Stdin:  nil,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	// Normally when elements differ diff returns a non-zero code
	_ = diffCmd.Run()
	return
}
