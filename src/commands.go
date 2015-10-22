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
	"errors"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cheggaaa/pb"
	"github.com/mattn/go-isatty"
	expirable "github.com/odeke-em/cache"
	"github.com/odeke-em/drive/config"
	"github.com/odeke-em/log"
)

var (
	ErrNoContext = errors.New("not in a drive context")
)

const (
	DriveIgnoreSuffix = ".driveignore"
)

type Options struct {
	// Depth is the number of pages/ listing recursion depth
	Depth int `cli:"depth"`
	// Exports contains the formats to export your Google Docs + Sheets to
	// e.g ["csv" "txt"]
	Exports []string `cli:"exports"`
	// ExportsDir is the directory to put the exported Google Docs + Sheets.
	// If not provided, will export them to the same dir as the source files are
	ExportsDir string `cli:"exports-dir"`
	// Force once set always converts NoChange into an Addition
	Force bool `cli:"force"`
	// Hidden discovers hidden paths if set
	Hidden       bool `cli:"hidden"`
	IgnoreRegexp *regexp.Regexp
	// IgnoreChecksum when set avoids the step
	// of comparing checksums as a final check.
	IgnoreChecksum bool `cli:"ignore"`
	// IgnoreConflict when set turns off the conflict resolution safety.
	IgnoreConflict bool `cli:"ignore-conflict"`
	// Allows listing of content in trash
	InTrash bool `cli:"in-trash"`
	Meta    *map[string][]string
	Mount   *config.Mount
	// NoClobber when set prevents overwriting of stale content
	NoClobber bool `cli:"no-clobber"`
	// NoPrompt overwrites any prompt pauses
	NoPrompt bool `cli:"no-prompt"`
	Path     string
	// PageSize determines the number of results returned per API call
	PageSize  int64 `cli:"pagesize"`
	Recursive bool  `cli:"recursive"`
	// Sources is a of list all paths that are
	// within the scope/path of the current gd context
	Sources []string `cli:"sources"`
	// TypeMask contains the result of setting different type bits e.g
	// Folder to search only for folders etc.
	TypeMask int `cli:"typeMask"`
	// Piped when set means to infer content to or from stdin
	Piped bool `cli:"piped"`
	// Quiet when set toggles only logging of errors to stderrs as
	// well as reading from stdin in this case stdout is not logged to
	Quiet             bool      `cli:"quiet"`
	StdoutIsTty       bool      `cli:"istty"`
	IgnoreNameClashes bool      `cli:"ignoreNameClashes"`
	ExcludeCrudMask   CrudValue `cli:"excludeCrudMask"`
	ExplicitlyExport  bool      `cli:"explicitlyExport"`
	Md5sum            bool      `cli:"md5sum"`
	indexingOnly      bool      `cli:"indexingOnly"`
	Verbose           bool      `cli:"verbose"`
	FixClashes        bool
}

type Commands struct {
	context *config.Context
	rem     *Remote
	opts    *Options
	rcOpts  *Options
	log     *log.Logger

	progress      *pb.ProgressBar
	mkdirAllCache *expirable.OperationCache
}

func (opts *Options) canPrompt() bool {
	if opts == nil || !opts.StdoutIsTty {
		return false
	}
	if opts.Quiet {
		return false
	}
	return !opts.NoPrompt
}

func (opts *Options) canPreview() bool {
	if opts == nil || !opts.StdoutIsTty {
		return false
	}
	if opts.Quiet {
		return false
	}
	return true
}

func rcPathChecker(absDir string) (string, error) {
	p := rcPath(absDir)
	statInfo, err := os.Stat(p)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if statInfo == nil {
		return "", os.ErrNotExist
	}
	return p, nil
}

func (opts *Options) rcPath() (string, error) {
	lastCurPath := ""
	for curPath := opts.Path; curPath != ""; curPath = path.Dir(curPath) {
		localRCP, err := rcPathChecker(curPath)
		if err == nil && localRCP != "" {
			return localRCP, nil
		}

		if false && err != nil && !os.IsNotExist(err) {
			return "", err
		}

		if lastCurPath == curPath { // Avoid getting a stalemate incase path.Dir cannot progress
			break
		}

		lastCurPath = curPath
	}

	return rcPathChecker(FsHomeDir)
}

func New(context *config.Context, opts *Options) *Commands {
	var r *Remote
	if context != nil {
		r = NewRemoteContext(context)
	}

	stdin, stdout, stderr := os.Stdin, os.Stdout, os.Stderr

	logger := log.New(stdin, stdout, stderr)

	if opts != nil {
		// should always start with /
		opts.Path = path.Clean(path.Join("/", opts.Path))

		if !opts.Force {
			ignoresPath := filepath.Join(context.AbsPath, DriveIgnoreSuffix)
			ignoreRegexp, regErr := combineIgnores(ignoresPath)

			if regErr != nil {
				logger.LogErrf("combining ignores from path %s and internally: %v\n", ignoresPath, regErr)
			}

			opts.IgnoreRegexp = ignoreRegexp
		}

		opts.StdoutIsTty = isatty.IsTerminal(stdout.Fd())

		if opts.Quiet {
			stdout = nil
		}
	}

	return &Commands{
		context:       context,
		rem:           r,
		opts:          opts,
		log:           logger,
		mkdirAllCache: expirable.New(),
	}
}

func combineIgnores(ignoresPath string) (*regexp.Regexp, error) {
	clauses, err := readCommentedFile(ignoresPath, "#")
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	clauses = append(clauses, internalIgnores()...)
	if len(clauses) < 1 {
		return nil, nil
	}

	regExComp, regErr := regexp.Compile(strings.Join(clauses, "|"))
	if regErr != nil {
		return nil, regErr
	}
	return regExComp, nil
}

func (g *Commands) taskStart(tasks int64) {
	if tasks > 0 {
		g.progress = newProgressBar(tasks)
	}
}

func newProgressBar(total int64) *pb.ProgressBar {
	pbf := pb.New64(total)
	pbf.Start()
	return pbf
}

func (g *Commands) taskAdd(n int64) {
	if g.progress != nil {
		g.progress.Add64(n)
	}
}

func (g *Commands) taskFinish() {
	if g.progress != nil {
		g.progress.Finish()
	}
}
