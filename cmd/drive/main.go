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

// Package contains the main entry point of gd.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/odeke-em/command" // branch -- track-visited
	"github.com/odeke-em/drive/config"
	"github.com/odeke-em/drive/gen"
	"github.com/odeke-em/drive/src"
)

var context *config.Context

func bindCommandWithAliases(key, description string, cmd command.Cmd, requiredFlags []string) {
	command.On(key, description, cmd, requiredFlags)
	aliases, ok := drive.Aliases[key]
	if ok {
		for _, alias := range aliases {
			command.On(alias, description, cmd, requiredFlags)
		}
	}
}

func translateKeyChecks(definedFlags map[string]*flag.Flag) map[string]bool {
	keysOnly := map[string]bool{}

	for k, _ := range definedFlags {
		keysOnly[k] = true
	}

	return keysOnly
}

func main() {
	maxProcs, err := strconv.ParseInt(os.Getenv(drive.GoMaxProcsKey), 10, 0)
	if err != nil || maxProcs < 1 {
		maxProcs = int64(drive.DefaultMaxProcs)
	}
	runtime.GOMAXPROCS(int(maxProcs))

	bindCommandWithAliases(drive.AboutKey, drive.DescAbout, &aboutCmd{}, []string{})
	bindCommandWithAliases(drive.CopyKey, drive.DescCopy, &copyCmd{}, []string{})
	bindCommandWithAliases(drive.DiffKey, drive.DescDiff, &diffCmd{}, []string{})
	bindCommandWithAliases(drive.EmptyTrashKey, drive.DescEmptyTrash, &emptyTrashCmd{}, []string{})
	bindCommandWithAliases(drive.FeaturesKey, drive.DescFeatures, &featuresCmd{}, []string{})
	bindCommandWithAliases(drive.InitKey, drive.DescInit, &initCmd{}, []string{})
	bindCommandWithAliases(drive.DeInitKey, drive.DescDeInit, &deInitCmd{}, []string{})
	bindCommandWithAliases(drive.HelpKey, drive.DescHelp, &helpCmd{}, []string{})

	bindCommandWithAliases(drive.ListKey, drive.DescList, &listCmd{}, []string{})
	bindCommandWithAliases(drive.MoveKey, drive.DescMove, &moveCmd{}, []string{})
	bindCommandWithAliases(drive.PullKey, drive.DescPull, &pullCmd{}, []string{})
	bindCommandWithAliases(drive.PushKey, drive.DescPush, &pushCmd{}, []string{})
	bindCommandWithAliases(drive.PubKey, drive.DescPublish, &publishCmd{}, []string{})
	bindCommandWithAliases(drive.RenameKey, drive.DescRename, &renameCmd{}, []string{})
	bindCommandWithAliases(drive.QuotaKey, drive.DescQuota, &quotaCmd{}, []string{})
	bindCommandWithAliases(drive.ShareKey, drive.DescShare, &shareCmd{}, []string{})
	bindCommandWithAliases(drive.StatKey, drive.DescStat, &statCmd{}, []string{})
	bindCommandWithAliases(drive.Md5sumKey, drive.DescMd5sum, &md5SumCmd{}, []string{})
	bindCommandWithAliases(drive.UnshareKey, drive.DescUnshare, &unshareCmd{}, []string{})
	bindCommandWithAliases(drive.TouchKey, drive.DescTouch, &touchCmd{}, []string{})
	bindCommandWithAliases(drive.TrashKey, drive.DescTrash, &trashCmd{}, []string{})
	bindCommandWithAliases(drive.UntrashKey, drive.DescUntrash, &untrashCmd{}, []string{})
	bindCommandWithAliases(drive.DeleteKey, drive.DescDelete, &deleteCmd{}, []string{})
	bindCommandWithAliases(drive.UnpubKey, drive.DescUnpublish, &unpublishCmd{}, []string{})
	bindCommandWithAliases(drive.VersionKey, drive.Version, &versionCmd{}, []string{})
	bindCommandWithAliases(drive.NewKey, drive.DescNew, &newCmd{}, []string{})
	bindCommandWithAliases(drive.IndexKey, drive.DescIndex, &indexCmd{}, []string{})
	bindCommandWithAliases(drive.UrlKey, drive.DescUrl, &urlCmd{}, []string{})
	bindCommandWithAliases(drive.OpenKey, drive.DescOpen, &openCmd{}, []string{})

	command.DefineHelp(&helpCmd{})
	command.ParseAndRun()
}

type helpCmd struct {
	args []string
}

func (cmd *helpCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	return fs
}

func (cmd *helpCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	drive.ShowDescriptions(args...)
	exitWithError(nil)
}

type featuresCmd struct{}

func (cmd *featuresCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	return fs
}

func (cmd *featuresCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	context, path := discoverContext(args)

	opts, _ := drive.ResourceConfigurationToOptions(path)
	// TODO: Log resource config errors if being verbose
	if opts == nil {
		opts = &drive.Options{Path: path}
	}

	exitWithError(drive.New(context, opts).About(drive.AboutFeatures))
}

type versionCmd struct{}

func (cmd *versionCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	return fs
}

func (cmd *versionCmd) Run(args []string) {
	drive.StdoutPrintf("drive version: %s\n%s\n", drive.Version, generated.PkgInfo)
	exitWithError(nil)
}

type initCmd struct{}

func (cmd *initCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	return fs
}

func (cmd *initCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	exitWithError(drive.New(initContext(args), nil).Init())
}

type deInitCmd struct {
	noPrompt *bool
}

func (cmd *deInitCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.noPrompt = fs.Bool(drive.NoPromptKey, false, "disables the prompt")
	return fs
}

func (cmd *deInitCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	_, context, path := preprocessArgsByToggle(args, true)
	opts := &drive.Options{
		NoPrompt: *cmd.noPrompt,
		Path:     path,
	}

	exitWithError(drive.New(context, opts).DeInit())
}

type quotaCmd struct{}

func (cmd *quotaCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	return fs
}

func (cmd *quotaCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	context, path := discoverContext(args)

	opts, _ := drive.ResourceConfigurationToOptions(path)
	// TODO: Log resource config errors if being verbose
	if opts == nil {
		opts = &drive.Options{}
	}

	opts.Path = path

	exitWithError(drive.New(context, opts).About(drive.AboutQuota))
}

type openCmd struct {
	byId    *bool
	local   *bool
	browser *bool
}

func (cmd *openCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "open by id instead of path")
	cmd.local = fs.Bool(drive.CLIOptionFileBrowser, true, "open file with the local file manager")
	cmd.browser = fs.Bool(drive.CLIOptionWebBrowser, true, "open file in default browser")
	return fs
}

func (cmd *openCmd) Run(args []string, definedArgs map[string]*flag.Flag) {
	sources, context, path := preprocessArgsByToggle(args, *cmd.byId)

	opts := drive.Options{
		Path:    path,
		Sources: sources,
	}

	openType := drive.OpenNone

	if *cmd.byId {
		openType |= drive.IdOpen
	}
	if *cmd.browser {
		openType |= drive.BrowserOpen
	}
	if *cmd.local {
		openType |= drive.FileManagerOpen
	}

	exitWithError(drive.New(context, &opts).Open(openType))
}

type urlCmd struct {
	byId *bool
}

func (cmd *urlCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "resolve url by id instead of path")
	return fs
}

func (cmd *urlCmd) Run(args []string, definedArgs map[string]*flag.Flag) {
	fmt.Println(definedArgs)
	sources, context, path := preprocessArgsByToggle(args, *cmd.byId)

	opts := drive.Options{
		Path:    path,
		Sources: sources,
	}

	exitWithError(drive.New(context, &opts).Url(*cmd.byId))
}

type listCmd struct {
	ById         *bool   `json:"by-id"`
	Hidden       *bool   `json:"hidden"`
	Recursive    *bool   `json:"recursive"`
	Files        *bool   `json:"files"`
	Directories  *bool   `json:"directories"`
	Depth        *int    `json:"depth"`
	PageSize     *int64  `json:"page-size"`
	LongFmt      *bool   `json:"long"`
	NoPrompt     *bool   `json:"no-prompt"`
	Shared       *bool   `json:"shared"`
	InTrash      *bool   `json:"in-trash"`
	Version      *bool   `json:"version"`
	Matches      *bool   `json:"matches"`
	Owners       *bool   `json:"owners"`
	Quiet        *bool   `json:"quiet"`
	SkipMimeKey  *string `json:"skip-mime"`
	MatchMimeKey *string `json:"match-mime"`
	ExactTitle   *string `json:"exact-title"`
	MatchOwner   *string `json:"match-owner"`
	ExactOwner   *string `json:"exact-owner"`
	NotOwner     *string `json:"not-owner"`
	Sort         *string `json:"sort"`
}

func (cmd *listCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.Depth = fs.Int(drive.DepthKey, 1, "maximum recursion depth")
	cmd.Hidden = fs.Bool(drive.HiddenKey, false, "list all paths even hidden ones")
	cmd.Files = fs.Bool(drive.CLIOptionFiles, false, "list only files")
	cmd.Directories = fs.Bool(drive.CLIOptionDirectories, false, "list all directories")
	cmd.LongFmt = fs.Bool(drive.CLIOptionLongFmt, false, "long listing of contents")
	cmd.PageSize = fs.Int64(drive.PageSizeKey, 100, "number of results per pagination")
	cmd.Shared = fs.Bool("shared", false, "show files that are shared with me")
	cmd.InTrash = fs.Bool(drive.TrashedKey, false, "list content in the trash")
	cmd.Version = fs.Bool("version", false, "show the number of times that the file has been modified on \n\t\tthe server even with changes not visible to the user")
	cmd.NoPrompt = fs.Bool(drive.NoPromptKey, false, "shows no prompt before pagination")
	cmd.Owners = fs.Bool("owners", false, "shows the owner names per file")
	cmd.Recursive = fs.Bool("r", false, "recursively list subdirectories")
	cmd.Sort = fs.String(drive.SortKey, "", drive.DescSort)
	cmd.Matches = fs.Bool(drive.MatchesKey, false, "list by prefix")
	cmd.Quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.SkipMimeKey = fs.String(drive.CLIOptionSkipMime, "", drive.DescSkipMime)
	cmd.MatchMimeKey = fs.String(drive.CLIOptionMatchMime, "", drive.DescMatchMime)
	cmd.ExactTitle = fs.String(drive.CLIOptionExactTitle, "", drive.DescExactTitle)
	cmd.MatchOwner = fs.String(drive.CLIOptionMatchOwner, "", drive.DescMatchOwner)
	cmd.ExactOwner = fs.String(drive.CLIOptionExactOwner, "", drive.DescExactOwner)
	cmd.NotOwner = fs.String(drive.CLIOptionNotOwner, "", drive.DescNotOwner)
	cmd.ById = fs.Bool(drive.CLIOptionId, false, "list by id instead of path")

	return fs
}

func (lCmd *listCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	fmt.Println("pk", definedFlags, args)

	for key, _ := range definedFlags {
		fmt.Println("key", key)
	}

	sources, context, path := preprocessArgsByToggle(args, (*lCmd.ById || *lCmd.Matches))
	parsed, err := drive.ResourceMappings(path)

	cmd := listCmd{}

	cs := drive.CliSifter{
		From:           *lCmd,
		Defaults:       parsed,
		AlreadyDefined: translateKeyChecks(definedFlags),
	}

	jsonStringified := drive.SiftCliTags(&cs)
	fmt.Println("jsonStringified", jsonStringified)
	if err := json.Unmarshal([]byte(jsonStringified), &cmd); err != nil {
		exitWithError(err)
	}

	fmt.Println(cmd.NotOwner)

	exitWithError(err)
	fmt.Println("parsed", parsed, "err", err, "passes", "virginCmd", cmd)

	typeMask := 0
	if *cmd.Directories {
		typeMask |= drive.Folder
	}
	if *cmd.Shared {
		typeMask |= drive.Shared
	}
	if *cmd.Owners {
		typeMask |= drive.Owners
	}
	if *cmd.Version {
		typeMask |= drive.CurrentVersion
	}
	if *cmd.Files {
		typeMask |= drive.NonFolder
	}
	if *cmd.InTrash {
		typeMask |= drive.InTrash
	}
	if !*cmd.LongFmt {
		typeMask |= drive.Minimal
	}

	opts, _ := drive.ResourceConfigurationToOptions(path)
	// TODO: Log resource config errors if being verbose
	if opts == nil {
		opts = &drive.Options{}
	}

	depth := *cmd.Depth
	if *cmd.Recursive {
		depth = drive.InfiniteDepth
	}

	meta := map[string][]string{
		drive.SortKey:         drive.NonEmptyTrimmedStrings(*cmd.Sort),
		drive.SkipMimeKeyKey:  drive.NonEmptyTrimmedStrings(strings.Split(*cmd.SkipMimeKey, ",")...),
		drive.MatchMimeKeyKey: drive.NonEmptyTrimmedStrings(strings.Split(*cmd.MatchMimeKey, ",")...),
		drive.ExactTitleKey:   drive.NonEmptyTrimmedStrings(strings.Split(*cmd.ExactTitle, ",")...),
		drive.MatchOwnerKey:   drive.NonEmptyTrimmedStrings(strings.Split(*cmd.MatchOwner, ",")...),
		drive.ExactOwnerKey:   drive.NonEmptyTrimmedStrings(strings.Split(*cmd.ExactOwner, ",")...),
		drive.NotOwnerKey:     drive.NonEmptyTrimmedStrings(strings.Split(*cmd.NotOwner, ",")...),
	}

	opts.Path = path
	if cmd.Hidden != nil {
		opts.Hidden = *cmd.Hidden
	}

	if cmd.Depth != nil {
		opts.Depth = *cmd.Depth
	}

	if cmd.InTrash != nil {
		opts.InTrash = *cmd.InTrash
	}

	if cmd.PageSize != nil {
		opts.PageSize = *cmd.PageSize
	}

	if cmd.NoPrompt != nil {
		opts.NoPrompt = *cmd.NoPrompt
	}

	if cmd.Recursive != nil {
		opts.Recursive = *cmd.Recursive
	}

	if cmd.Quiet != nil {
		opts.Quiet = *cmd.Quiet
	}

	options := drive.Options{
		Depth:     depth,
		Hidden:    *cmd.Hidden,
		InTrash:   *cmd.InTrash,
		PageSize:  *cmd.PageSize,
		Path:      path,
		NoPrompt:  *cmd.NoPrompt,
		Recursive: *cmd.Recursive,
		Sources:   sources,
		TypeMask:  typeMask,
		Quiet:     *cmd.Quiet,
		Meta:      &meta,
	}

	fmt.Println("options", options)

	opts.Sources = sources
	opts.TypeMask = typeMask

	keysOnly := translateKeyChecks(definedFlags)
	defaultOpts, err := drive.ResourceConfigurationToOptions(path)

	if err == nil && defaultOpts != nil {
		drive.CopyOptionsFromKeysIfNotSet(defaultOpts, opts, keysOnly)
	}

	if *cmd.Shared {
		exitWithError(drive.New(context, opts).ListShared())
	} else if *cmd.Matches {
		exitWithError(drive.New(context, opts).ListMatches())
	} else {
		exitWithError(drive.New(context, opts).List(*cmd.ById))
	}
}

type md5SumCmd struct {
	byId      *bool
	depth     *int
	hidden    *bool
	recursive *bool
	quiet     *bool
}

func (cmd *md5SumCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.depth = fs.Int(drive.DepthKey, 1, "max traversal depth")
	cmd.hidden = fs.Bool(drive.HiddenKey, false, "discover hidden paths")
	cmd.recursive = fs.Bool("r", false, "recursively discover folders")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "stat by id instead of path")
	return fs
}

func (cmd *md5SumCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgsByToggle(args, *cmd.byId)

	depth := *cmd.depth
	if *cmd.recursive {
		depth = drive.InfiniteDepth
	}

	opts := drive.Options{
		Hidden:    *cmd.hidden,
		Path:      path,
		Recursive: *cmd.recursive,
		Sources:   sources,
		Quiet:     *cmd.quiet,
		Depth:     depth,
		Md5sum:    true,
	}

	if *cmd.byId {
		exitWithError(drive.New(context, &opts).StatById())
	} else {
		exitWithError(drive.New(context, &opts).Stat())
	}
}

type statCmd struct {
	byId      *bool
	depth     *int
	hidden    *bool
	recursive *bool
	quiet     *bool
	md5sum    *bool
}

func (cmd *statCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.depth = fs.Int(drive.DepthKey, 1, "max traversal depth")
	cmd.hidden = fs.Bool(drive.HiddenKey, false, "discover hidden paths")
	cmd.recursive = fs.Bool("r", false, "recursively discover folders")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "stat by id instead of path")
	cmd.md5sum = fs.Bool(drive.Md5sumKey, false, "produce output compatible with md5sum(1)")
	return fs
}

func (cmd *statCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgsByToggle(args, *cmd.byId)

	depth := *cmd.depth
	if *cmd.recursive {
		depth = drive.InfiniteDepth
	}

	opts := drive.Options{
		Hidden:    *cmd.hidden,
		Path:      path,
		Recursive: *cmd.recursive,
		Sources:   sources,
		Quiet:     *cmd.quiet,
		Depth:     depth,
		Md5sum:    *cmd.md5sum,
	}

	if *cmd.byId {
		exitWithError(drive.New(context, &opts).StatById())
	} else {
		exitWithError(drive.New(context, &opts).Stat())
	}
}

type indexCmd struct {
	byId              *bool
	ignoreConflict    *bool
	recursive         *bool
	noPrompt          *bool
	hidden            *bool
	force             *bool
	ignoreNameClashes *bool
	quiet             *bool
	excludeOps        *string
	skipMimeKey       *string
	ignoreChecksum    *bool
	noClobber         *bool
	prune             *bool
	allOps            *bool
	matches           *bool
}

func (cmd *indexCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "fetch by id instead of path")
	cmd.ignoreConflict = fs.Bool(drive.CLIOptionIgnoreConflict, true, drive.DescIgnoreConflict)
	cmd.recursive = fs.Bool("r", true, "fetch recursively for children")
	cmd.noPrompt = fs.Bool(drive.NoPromptKey, false, "shows no prompt before applying the fetch action")
	cmd.hidden = fs.Bool(drive.HiddenKey, true, "allows fetching of hidden paths")
	cmd.force = fs.Bool(drive.ForceKey, false, "forces a fetch even if no changes present")
	cmd.ignoreNameClashes = fs.Bool(drive.CLIOptionIgnoreNameClashes, true, drive.DescIgnoreNameClashes)
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.excludeOps = fs.String(drive.CLIOptionExcludeOperations, "", drive.DescExcludeOps)
	cmd.skipMimeKey = fs.String(drive.CLIOptionSkipMime, "", drive.DescSkipMime)
	cmd.ignoreChecksum = fs.Bool(drive.CLIOptionIgnoreChecksum, true, drive.DescIgnoreChecksum)
	cmd.noClobber = fs.Bool(drive.CLIOptionNoClobber, false, "prevents overwriting of old content")
	cmd.prune = fs.Bool(drive.CLIOptionPruneIndices, false, drive.DescPruneIndices)
	cmd.allOps = fs.Bool(drive.CLIOptionAllIndexOperations, false, drive.DescAllIndexOperations)
	cmd.matches = fs.Bool(drive.MatchesKey, false, "search by prefix")

	return fs
}

type errorer func() error

func (cmd *indexCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	byId := *cmd.byId
	byMatches := *cmd.matches
	sources, context, path := preprocessArgsByToggle(args, byMatches || byId)

	options := &drive.Options{
		Sources:           sources,
		Hidden:            *cmd.hidden,
		IgnoreChecksum:    *cmd.ignoreChecksum,
		IgnoreConflict:    *cmd.ignoreConflict,
		NoPrompt:          *cmd.noPrompt,
		NoClobber:         *cmd.noClobber,
		Path:              path,
		Recursive:         *cmd.recursive,
		Quiet:             *cmd.quiet,
		Force:             *cmd.force,
		IgnoreNameClashes: *cmd.ignoreNameClashes,
	}

	dr := drive.New(context, options)

	fetchFn := dr.Fetch
	if byId {
		fetchFn = dr.FetchById
	} else if *cmd.matches {
		fetchFn = dr.FetchMatches
	}

	scheduling := []errorer{}
	if *cmd.allOps {
		scheduling = append(scheduling, dr.Prune, fetchFn)
	} else if *cmd.prune {
		scheduling = append(scheduling, dr.Prune)
	} else {
		scheduling = append(scheduling, fetchFn)
	}

	for _, fn := range scheduling {
		exitWithError(fn())
	}
}

type pullCmd struct {
	byId              *bool
	exportsDir        *string
	export            *string
	excludeOps        *string
	force             *bool
	hidden            *bool
	matches           *bool
	noPrompt          *bool
	noClobber         *bool
	recursive         *bool
	ignoreChecksum    *bool
	ignoreConflict    *bool
	piped             *bool
	quiet             *bool
	ignoreNameClashes *bool
	skipMimeKey       *string
	explicitlyExport  *bool
	fixClashes        *bool

	verbose *bool
	depth   *int
}

func (cmd *pullCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.noClobber = fs.Bool(drive.CLIOptionNoClobber, false, "prevents overwriting of old content")
	cmd.export = fs.String(
		"export", "", "comma separated list of formats to export your docs + sheets files")
	cmd.recursive = fs.Bool("r", true, "performs the pull action recursively")
	cmd.noPrompt = fs.Bool(drive.NoPromptKey, false, "shows no prompt before applying the pull action")
	cmd.hidden = fs.Bool(drive.HiddenKey, false, "allows pulling of hidden paths")
	cmd.force = fs.Bool(drive.ForceKey, false, "forces a pull even if no changes present")
	cmd.ignoreChecksum = fs.Bool(drive.CLIOptionIgnoreChecksum, true, drive.DescIgnoreChecksum)
	cmd.ignoreConflict = fs.Bool(drive.CLIOptionIgnoreConflict, false, drive.DescIgnoreConflict)
	cmd.ignoreNameClashes = fs.Bool(drive.CLIOptionIgnoreNameClashes, false, drive.DescIgnoreNameClashes)
	cmd.exportsDir = fs.String("export-dir", "", "directory to place exports")
	cmd.matches = fs.Bool(drive.MatchesKey, false, "search by prefix")
	cmd.piped = fs.Bool("piped", false, "if true, read content from stdin")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.excludeOps = fs.String(drive.CLIOptionExcludeOperations, "", drive.DescExcludeOps)
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "pull by id instead of path")
	cmd.skipMimeKey = fs.String(drive.CLIOptionSkipMime, "", drive.DescSkipMime)
	cmd.explicitlyExport = fs.Bool(drive.CLIOptionExplicitlyExport, false, drive.DescExplicitylPullExports)
	cmd.verbose = fs.Bool(drive.CLIOptionVerboseKey, false, drive.DescVerbose)
	cmd.depth = fs.Int(drive.DepthKey, drive.DefaultMaxTraversalDepth, "max traversal depth")
	cmd.fixClashes = fs.Bool(drive.CLIOptionFixClashesKey, false, drive.DescFixClashes)

	return fs
}

func (cmd *pullCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgsByToggle(args, (*cmd.byId || *cmd.matches))

	excludes := drive.NonEmptyTrimmedStrings(strings.Split(*cmd.excludeOps, ",")...)
	excludeCrudMask := drive.CrudAtoi(excludes...)
	if excludeCrudMask == drive.AllCrudOperations {
		exitWithError(fmt.Errorf("all CRUD operations forbidden"))
	}

	meta := map[string][]string{
		drive.SkipMimeKeyKey: drive.NonEmptyTrimmedStrings(strings.Split(*cmd.skipMimeKey, ",")...),
	}

	// Filter out empty strings.
	exports := drive.NonEmptyTrimmedStrings(strings.Split(*cmd.export, ",")...)

	options := &drive.Options{
		Exports:           uniqOrderedStr(exports),
		ExportsDir:        strings.Trim(*cmd.exportsDir, " "),
		Force:             *cmd.force,
		Hidden:            *cmd.hidden,
		IgnoreChecksum:    *cmd.ignoreChecksum,
		IgnoreConflict:    *cmd.ignoreConflict,
		NoPrompt:          *cmd.noPrompt,
		NoClobber:         *cmd.noClobber,
		Path:              path,
		Recursive:         *cmd.recursive,
		Sources:           sources,
		Piped:             *cmd.piped,
		Quiet:             *cmd.quiet,
		IgnoreNameClashes: *cmd.ignoreNameClashes,
		ExcludeCrudMask:   excludeCrudMask,
		ExplicitlyExport:  *cmd.explicitlyExport,
		Meta:              &meta,
		Verbose:           *cmd.verbose,
		Depth:             *cmd.depth,
		FixClashes:        *cmd.fixClashes,
	}

	if *cmd.matches {
		exitWithError(drive.New(context, options).PullMatches())
	} else if *cmd.piped {
		exitWithError(drive.New(context, options).PullPiped(*cmd.byId))
	} else {
		exitWithError(drive.New(context, options).Pull(*cmd.byId))
	}
}

type pushCmd struct {
	noClobber   *bool
	hidden      *bool
	force       *bool
	noPrompt    *bool
	recursive   *bool
	piped       *bool
	mountedPush *bool
	// convert when set tells Google drive to convert the document into
	// its appropriate Google Docs format
	convert *bool
	// ocr when set indicates that Optical Character Recognition should be
	// attempted on .[gif, jpg, pdf, png] uploads
	ocr               *bool
	ignoreChecksum    *bool
	ignoreConflict    *bool
	ignoreNameClashes *bool
	quiet             *bool
	coercedMimeKey    *string
	excludeOps        *string
	skipMimeKey       *string
	verbose           *bool
	depth             *int
	fixClashes        *bool
}

func (cmd *pushCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.noClobber = fs.Bool(drive.CLIOptionNoClobber, false, "allows overwriting of old content")
	cmd.hidden = fs.Bool(drive.HiddenKey, false, "allows pushing of hidden paths")
	cmd.recursive = fs.Bool("r", true, "performs the push action recursively")
	cmd.noPrompt = fs.Bool(drive.NoPromptKey, false, "shows no prompt before applying the push action")
	cmd.force = fs.Bool(drive.ForceKey, false, "forces a push even if no changes present")
	cmd.mountedPush = fs.Bool("m", false, "allows pushing of mounted paths")
	cmd.convert = fs.Bool("convert", false, "toggles conversion of the file to its appropriate Google Doc format")
	cmd.ocr = fs.Bool("ocr", false, "if true, attempt OCR on gif, jpg, pdf and png uploads")
	cmd.piped = fs.Bool("piped", false, "if true, read content from stdin")
	cmd.ignoreChecksum = fs.Bool(drive.CLIOptionIgnoreChecksum, true, drive.DescIgnoreChecksum)
	cmd.ignoreConflict = fs.Bool(drive.CLIOptionIgnoreConflict, false, drive.DescIgnoreConflict)
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.coercedMimeKey = fs.String(drive.CoercedMimeKeyKey, "", "the mimeType you are trying to coerce this file to be")
	cmd.ignoreNameClashes = fs.Bool(drive.CLIOptionIgnoreNameClashes, false, drive.DescIgnoreNameClashes)
	cmd.excludeOps = fs.String(drive.CLIOptionExcludeOperations, "", drive.DescExcludeOps)
	cmd.skipMimeKey = fs.String(drive.CLIOptionSkipMime, "", drive.DescSkipMime)
	cmd.verbose = fs.Bool(drive.CLIOptionVerboseKey, false, drive.DescVerbose)
	cmd.depth = fs.Int(drive.DepthKey, drive.DefaultMaxTraversalDepth, "max traversal depth")
	cmd.fixClashes = fs.Bool(drive.CLIOptionFixClashesKey, false, drive.DescFixClashes)
	return fs
}

func (cmd *pushCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	fmt.Println("definedFlags", definedFlags)
	if *cmd.mountedPush {
		cmd.pushMounted(args, definedFlags)
	} else {
		sources, context, path := preprocessArgs(args)

		options := cmd.createPushOptions()
		options.Path = path
		options.Sources = sources
		options.FixClashes = *cmd.fixClashes

		keysOnly := translateKeyChecks(definedFlags)
		defaultOpts, err := drive.ResourceConfigurationToOptions(path)

		if err == nil && defaultOpts != nil {
			drive.CopyOptionsFromKeysIfNotSet(defaultOpts, options, keysOnly)
		}

		if *cmd.piped {
			exitWithError(drive.New(context, options).PushPiped())
		} else {
			exitWithError(drive.New(context, options).Push())
		}
	}
}

type touchCmd struct {
	byId      *bool
	hidden    *bool
	recursive *bool
	matches   *bool
	quiet     *bool
}

func (cmd *touchCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool(drive.HiddenKey, false, "allows pushing of hidden paths")
	cmd.recursive = fs.Bool("r", false, "toggles recursive touching")
	cmd.matches = fs.Bool(drive.MatchesKey, false, "search by prefix and touch")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "share by id instead of path")
	return fs
}

func (cmd *touchCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgsByToggle(args, *cmd.matches || *cmd.byId)

	opts := drive.Options{
		Hidden:    *cmd.hidden,
		Path:      path,
		Recursive: *cmd.recursive,
		Sources:   sources,
		Quiet:     *cmd.quiet,
	}

	if *cmd.matches {
		exitWithError(drive.New(context, &opts).TouchByMatch())
	} else {
		exitWithError(drive.New(context, &opts).Touch(*cmd.byId))
	}
}

func (cmd *pushCmd) createPushOptions() *drive.Options {
	mask := drive.OptNone
	if *cmd.convert {
		mask |= drive.OptConvert
	}
	if *cmd.ocr {
		mask |= drive.OptOCR
	}

	meta := map[string][]string{
		drive.CoercedMimeKeyKey: drive.NonEmptyTrimmedStrings(*cmd.coercedMimeKey),
		drive.SkipMimeKeyKey:    drive.NonEmptyTrimmedStrings(strings.Split(*cmd.skipMimeKey, ",")...),
	}

	excludes := drive.NonEmptyTrimmedStrings(strings.Split(*cmd.excludeOps, ",")...)
	excludeCrudMask := drive.CrudAtoi(excludes...)
	if excludeCrudMask == drive.AllCrudOperations {
		exitWithError(fmt.Errorf("all CRUD operations forbidden yet asking to push"))
	}

	return &drive.Options{
		Force:             *cmd.force,
		Hidden:            *cmd.hidden,
		IgnoreChecksum:    *cmd.ignoreChecksum,
		IgnoreConflict:    *cmd.ignoreConflict,
		NoClobber:         *cmd.noClobber,
		NoPrompt:          *cmd.noPrompt,
		Recursive:         *cmd.recursive,
		Piped:             *cmd.piped,
		Quiet:             *cmd.quiet,
		Meta:              &meta,
		TypeMask:          mask,
		ExcludeCrudMask:   excludeCrudMask,
		IgnoreNameClashes: *cmd.ignoreNameClashes,
		Verbose:           *cmd.verbose,
		Depth:             *cmd.depth,
	}
}

func (cmd *pushCmd) pushMounted(args []string, definedFlags map[string]*flag.Flag) {
	argc := len(args)

	var contextArgs, rest, sources []string

	if !*cmd.mountedPush {
		contextArgs = args
	} else {
		// Expectation is that at least one path has to be passed in
		if argc < 2 {
			cwd, cerr := os.Getwd()
			if cerr != nil {
				contextArgs = []string{cwd}
			}
			rest = args
		} else {
			rest = args[:argc-1]
			contextArgs = args[argc-1:]
		}
	}

	rest = drive.NonEmptyStrings(rest...)
	context, path := discoverContext(contextArgs)

	contextAbsPath, err := filepath.Abs(path)
	exitWithError(err)

	if path == "." {
		path = ""
	}

	mount, auxSrcs := config.MountPoints(path, contextAbsPath, rest, *cmd.hidden)

	root := context.AbsPathOf("")

	sources, err = relativePathsOpt(root, auxSrcs, true)
	exitWithError(err)

	options := cmd.createPushOptions()
	options.Mount = mount
	options.Sources = sources

	keysOnly := translateKeyChecks(definedFlags)
	defaultOpts, err := drive.ResourceConfigurationToOptions(path)

	if err == nil && defaultOpts != nil {
		drive.CopyOptionsFromKeysIfNotSet(defaultOpts, options, keysOnly)
	}

	exitWithError(drive.New(context, options).Push())
}

type aboutCmd struct {
	features *bool
	quota    *bool
	filesize *bool
	quiet    *bool
}

func (cmd *aboutCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.features = fs.Bool("features", false, "gives information on features present on this drive")
	cmd.quota = fs.Bool("quota", false, "prints out quota information for this drive")
	cmd.filesize = fs.Bool("filesize", false, "prints out information about file sizes e.g the max upload size for a specific file size")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	return fs
}

func (cmd *aboutCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	_, context, _ := preprocessArgs(args)

	mask := drive.AboutNone
	if *cmd.features {
		mask |= drive.AboutFeatures
	}
	if *cmd.quota {
		mask |= drive.AboutQuota
	}
	if *cmd.filesize {
		mask |= drive.AboutFileSizes
	}

	if mask == drive.AboutNone { // No option set
		mask = drive.AboutQuota | drive.AboutFeatures | drive.AboutFileSizes
	}

	exitWithError(drive.New(context, &drive.Options{
		Quiet: *cmd.quiet,
	}).About(mask))
}

type diffCmd struct {
	hidden            *bool
	ignoreConflict    *bool
	ignoreChecksum    *bool
	ignoreNameClashes *bool
	quiet             *bool
	depth             *int
}

func (cmd *diffCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool(drive.HiddenKey, false, "allows pulling of hidden paths")
	cmd.ignoreChecksum = fs.Bool(drive.CLIOptionIgnoreChecksum, true, drive.DescIgnoreChecksum)
	cmd.ignoreConflict = fs.Bool(drive.CLIOptionIgnoreConflict, false, drive.DescIgnoreConflict)
	cmd.ignoreNameClashes = fs.Bool(drive.CLIOptionIgnoreNameClashes, false, drive.DescIgnoreNameClashes)
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.depth = fs.Int(drive.DepthKey, drive.DefaultMaxTraversalDepth, "max traversal depth")
	return fs
}

func (cmd *diffCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgs(args)

	exitWithError(drive.New(context, &drive.Options{
		Recursive:         true,
		Path:              path,
		Hidden:            *cmd.hidden,
		Sources:           sources,
		IgnoreChecksum:    *cmd.ignoreChecksum,
		IgnoreNameClashes: *cmd.ignoreNameClashes,
		IgnoreConflict:    *cmd.ignoreConflict,
		Quiet:             *cmd.quiet,
		Depth:             *cmd.depth,
	}).Diff())
}

type publishCmd struct {
	hidden *bool
	quiet  *bool
	byId   *bool
}

type unpublishCmd struct {
	hidden *bool
	quiet  *bool
	byId   *bool
}

func (cmd *unpublishCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool(drive.HiddenKey, false, "allows pulling of hidden paths")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "unpublish by id instead of path")
	return fs
}

func (cmd *unpublishCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgsByToggle(args, *cmd.byId)
	exitWithError(drive.New(context, &drive.Options{
		Path:    path,
		Sources: sources,
		Quiet:   *cmd.quiet,
	}).Unpublish(*cmd.byId))
}

type emptyTrashCmd struct {
	noPrompt *bool
	quiet    *bool
}

func (cmd *emptyTrashCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.noPrompt = fs.Bool(drive.NoPromptKey, false, "shows no prompt before emptying the trash")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	return fs
}

func (cmd *emptyTrashCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	_, context, _ := preprocessArgs(args)
	exitWithError(drive.New(context, &drive.Options{
		NoPrompt: *cmd.noPrompt,
		Quiet:    *cmd.quiet,
	}).EmptyTrash())
}

type deleteCmd struct {
	hidden  *bool
	matches *bool
	quiet   *bool
	byId    *bool
}

func (cmd *deleteCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool(drive.HiddenKey, false, "allows trashing hidden paths")
	cmd.matches = fs.Bool(drive.MatchesKey, false, "search by prefix and delete")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "delete by id instead of path")
	return fs
}

func (cmd *deleteCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgsByToggle(args, *cmd.matches || *cmd.byId)

	opts := drive.Options{
		Path:    path,
		Sources: sources,
		Quiet:   *cmd.quiet,
	}

	if !*cmd.matches {
		exitWithError(drive.New(context, &opts).Delete(*cmd.byId))
	} else {
		exitWithError(drive.New(context, &opts).DeleteByMatch())
	}
}

type trashCmd struct {
	hidden  *bool
	matches *bool
	quiet   *bool
	byId    *bool
}

func (cmd *trashCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool(drive.HiddenKey, false, "allows trashing hidden paths")
	cmd.matches = fs.Bool(drive.MatchesKey, false, "search by prefix and trash")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "trash by id instead of path")
	return fs
}

func (cmd *trashCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgsByToggle(args, *cmd.matches || *cmd.byId)

	opts := drive.Options{
		Path:    path,
		Sources: sources,
		Quiet:   *cmd.quiet,
	}

	if !*cmd.matches {
		exitWithError(drive.New(context, &opts).Trash(*cmd.byId))
	} else {
		exitWithError(drive.New(context, &opts).TrashByMatch())
	}
}

type newCmd struct {
	folder  *bool
	mimeKey *string
}

func (cmd *newCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.folder = fs.Bool("folder", false, "create a folder if set otherwise create a regular file")
	cmd.mimeKey = fs.String(drive.MimeKey, "", "coerce the file to this mimeType")
	return fs
}

func (cmd *newCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgs(args)

	opts := drive.Options{
		Path:    path,
		Sources: sources,
	}

	meta := map[string][]string{
		drive.MimeKey: drive.NonEmptyTrimmedStrings(strings.Split(*cmd.mimeKey, ",")...),
	}

	opts.Meta = &meta

	if *cmd.folder {
		exitWithError(drive.New(context, &opts).NewFolder())
	} else {
		exitWithError(drive.New(context, &opts).NewFile())
	}
}

type copyCmd struct {
	quiet     *bool
	recursive *bool
	byId      *bool
}

func (cmd *copyCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.recursive = fs.Bool("r", false, "recursive copying")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "copy by id instead of path")
	return fs
}

func (cmd *copyCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	if len(args) < 2 {
		args = append(args, ".")
	}

	end := len(args) - 1
	if end < 1 {
		exitWithError(fmt.Errorf("copy: expected more than one path"))
	}

	dest := args[end]

	sources, context, path := preprocessArgsByToggle(args, *cmd.byId)

	// Unshift by the end path
	sources = sources[:len(sources)-1]
	destRels, err := relativePaths(context.AbsPathOf(""), dest)
	exitWithError(err)

	dest = destRels[0]
	sources = append(sources, dest)

	exitWithError(drive.New(context, &drive.Options{
		Path:      path,
		Sources:   sources,
		Recursive: *cmd.recursive,
		Quiet:     *cmd.quiet,
	}).Copy(*cmd.byId))
}

type untrashCmd struct {
	hidden  *bool
	matches *bool
	quiet   *bool
	byId    *bool
}

func (cmd *untrashCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool(drive.HiddenKey, false, "allows untrashing hidden paths")
	cmd.matches = fs.Bool(drive.MatchesKey, false, "search by prefix and untrash")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "untrash by id instead of path")
	return fs
}

func (cmd *untrashCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgsByToggle(args, *cmd.byId || *cmd.matches)

	opts := drive.Options{
		Path:    path,
		Sources: sources,
		Quiet:   *cmd.quiet,
	}

	if !*cmd.matches {
		exitWithError(drive.New(context, &opts).Untrash(*cmd.byId))
	} else {
		exitWithError(drive.New(context, &opts).UntrashByMatch())
	}
}

func (cmd *publishCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool(drive.HiddenKey, false, "allows publishing of hidden paths")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "publish by id instead of path")
	return fs
}

func (cmd *publishCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgsByToggle(args, *cmd.byId)
	exitWithError(drive.New(context, &drive.Options{
		Path:    path,
		Sources: sources,
		Quiet:   *cmd.quiet,
	}).Publish(*cmd.byId))
}

type unshareCmd struct {
	noPrompt    *bool
	accountType *string
	quiet       *bool
	byId        *bool
}

func (cmd *unshareCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.accountType = fs.String(drive.TypeKey, "", "scope of account to revoke access to")
	cmd.noPrompt = fs.Bool(drive.NoPromptKey, false, "disables the prompt")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "unshare by id instead of path")
	return fs
}

func (cmd *unshareCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgsByToggle(args, *cmd.byId)

	meta := map[string][]string{
		"accountType": uniqOrderedStr(drive.NonEmptyTrimmedStrings(strings.Split(*cmd.accountType, ",")...)),
	}

	exitWithError(drive.New(context, &drive.Options{
		Meta:     &meta,
		Path:     path,
		Sources:  sources,
		NoPrompt: *cmd.noPrompt,
		Quiet:    *cmd.quiet,
	}).Unshare(*cmd.byId))
}

type moveCmd struct {
	quiet *bool
	byId  *bool
}

func (cmd *moveCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "move by id instead of path")
	return fs
}

func (cmd *moveCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	argc := len(args)
	if argc < 1 {
		exitWithError(fmt.Errorf("move: expecting a path or more"))
	}
	sources, context, path := preprocessArgsByToggle(args, *cmd.byId)

	// Unshift by the end path
	sources = sources[:len(sources)-1]

	dest := args[argc-1]
	destRels, err := relativePaths(context.AbsPathOf(""), dest)
	exitWithError(err)

	sources = append(sources, destRels[0])

	exitWithError(drive.New(context, &drive.Options{
		Path:    path,
		Sources: sources,
		Quiet:   *cmd.quiet,
	}).Move(*cmd.byId))
}

type renameCmd struct {
	force *bool
	quiet *bool
	byId  *bool
}

func (cmd *renameCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.force = fs.Bool(drive.ForceKey, false, "coerce rename even if remote already exists")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "unshare by id instead of path")
	return fs
}

func (cmd *renameCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	argc := len(args)
	if argc < 2 {
		exitWithError(fmt.Errorf("rename: expecting <src> <dest>"))
	}
	rest, last := args[:argc-1], args[argc-1]
	sources, context, path := preprocessArgsByToggle(rest, *cmd.byId)

	sources = append(sources, last)
	exitWithError(drive.New(context, &drive.Options{
		Path:    path,
		Sources: sources,
		Force:   *cmd.force,
		Quiet:   *cmd.quiet,
	}).Rename(*cmd.byId))
}

type shareCmd struct {
	byId        *bool
	emails      *string
	message     *string
	role        *string
	accountType *string
	noPrompt    *bool
	notify      *bool
	quiet       *bool
}

func (cmd *shareCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.emails = fs.String(drive.EmailsKey, "", "emails to share the file to")
	cmd.message = fs.String("message", "", "message to send receipients")
	cmd.role = fs.String(drive.RoleKey, "", "role to set to receipients of share. Possible values: "+drive.DescRoles)
	cmd.accountType = fs.String(drive.TypeKey, "", "scope of accounts to share files with. Possible values: "+drive.DescAccountTypes)
	cmd.notify = fs.Bool(drive.CLIOptionNotify, true, "toggle whether to notify receipients about share")
	cmd.noPrompt = fs.Bool(drive.NoPromptKey, false, "disables the prompt")
	cmd.quiet = fs.Bool(drive.QuietKey, false, "if set, do not log anything but errors")
	cmd.byId = fs.Bool(drive.CLIOptionId, false, "share by id instead of path")
	return fs
}

func (cmd *shareCmd) Run(args []string, definedFlags map[string]*flag.Flag) {
	sources, context, path := preprocessArgsByToggle(args, *cmd.byId)

	meta := map[string][]string{
		drive.EmailMessageKey: []string{*cmd.message},
		drive.EmailsKey:       uniqOrderedStr(drive.NonEmptyTrimmedStrings(strings.Split(*cmd.emails, ",")...)),
		drive.RoleKey:         uniqOrderedStr(drive.NonEmptyTrimmedStrings(strings.Split(*cmd.role, ",")...)),
		"accountType":         uniqOrderedStr(drive.NonEmptyTrimmedStrings(strings.Split(*cmd.accountType, ",")...)),
	}

	mask := drive.NoopOnShare
	if *cmd.notify {
		mask = drive.Notify
	}

	exitWithError(drive.New(context, &drive.Options{
		Meta:     &meta,
		Path:     path,
		Sources:  sources,
		TypeMask: mask,
		NoPrompt: *cmd.noPrompt,
		Quiet:    *cmd.quiet,
	}).Share(*cmd.byId))
}

func initContext(args []string) *config.Context {
	var err error
	var gdPath string
	var firstInit bool

	gdPath, firstInit, context, err = config.Initialize(getContextPath(args))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// The signal handler should clean up the .gd path if this is the first time
	go func() {
		_ = <-c
		if firstInit {
			os.RemoveAll(gdPath)
		}
		os.Exit(1)
	}()

	exitWithError(err)
	return context
}

func discoverContext(args []string) (*config.Context, string) {
	var err error
	context, err = config.Discover(getContextPath(args))
	exitWithError(err)
	relPath := ""
	if len(args) > 0 {
		var headAbsArg string
		headAbsArg, err = filepath.Abs(args[0])
		if err == nil {
			relPath, err = filepath.Rel(context.AbsPath, headAbsArg)
		}
	}

	exitWithError(err)

	// relPath = strings.Join([]string{"", relPath}, "/")
	return context, relPath
}

func getContextPath(args []string) (contextPath string) {
	if len(args) > 0 {
		contextPath, _ = filepath.Abs(args[0])
	}
	if contextPath == "" {
		contextPath, _ = os.Getwd()
	}
	return
}

func uniqOrderedStr(sources []string) []string {
	cache := map[string]bool{}
	var uniqPaths []string
	for _, p := range sources {
		ok := cache[p]
		if ok {
			continue
		}
		uniqPaths = append(uniqPaths, p)
		cache[p] = true
	}
	return uniqPaths
}

func exitWithError(err error) {
	if err != nil {
		drive.FprintfShadow(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func relativePaths(root string, args ...string) ([]string, error) {
	return relativePathsOpt(root, args, false)
}

func relativePathsOpt(root string, args []string, leastNonExistant bool) ([]string, error) {
	var err error
	var relPath string
	var relPaths []string

	for _, p := range args {
		p, err = filepath.Abs(p)
		if err != nil {
			drive.FprintfShadow(os.Stderr, "%s %v\n", p, err)
			continue
		}

		if leastNonExistant {
			sRoot := config.LeastNonExistantRoot(p)
			if sRoot != "" {
				p = sRoot
			}
		}

		relPath, err = filepath.Rel(root, p)
		if err != nil {
			break
		}

		if relPath == "." {
			relPath = ""
		}

		relPath = "/" + relPath
		relPaths = append(relPaths, relPath)
	}

	return relPaths, err
}

func preprocessArgs(args []string) ([]string, *config.Context, string) {
	context, path := discoverContext(args)
	root := context.AbsPathOf("")

	if len(args) < 1 {
		args = []string{"."}
	}

	relPaths, err := relativePaths(root, args...)
	exitWithError(err)

	return uniqOrderedStr(relPaths), context, path
}

func preprocessArgsByToggle(args []string, skipArgPreprocess bool) (sources []string, context *config.Context, path string) {
	if !skipArgPreprocess {
		return preprocessArgs(args)
	}

	cwd, err := os.Getwd()
	exitWithError(err)

	_, context, path = preprocessArgs([]string{cwd})
	sources = uniqOrderedStr(args)
	return sources, context, path
}
