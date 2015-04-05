package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

// ExtractFileExtension extracts the extension from a filename. This is defined
// as the remainder of the string after the last '.'.
func ExtractFileExtension(filename string) string {
	patternPermission := regexp.MustCompile(`.*\.(.*):.*`)
	permissionMatch := patternPermission.FindStringSubmatch(filename)
	if len(permissionMatch) > 0 {
		return permissionMatch[1]
	}
	patternFilename := regexp.MustCompile(`.*\.(.*)`)
	return patternFilename.FindStringSubmatch(filename)[1]
}

// LookupExtensionByImage is a closure storing a dictionary mapping source
// extensions to the names of Docker Exec images.
var LookupExtensionByImage = func() func(string) string {
	innerMap := map[string]string{
		"c":      "c",
		"clj":    "clojure",
		"coffee": "coffee",
		"cpp":    "cpp",
		"cs":     "csharp",
		"d":      "d",
		"erl":    "erlang",
		"fs":     "fsharp",
		"go":     "go",
		"groovy": "groovy",
		"hs":     "haskell",
		"java":   "java",
		"lisp":   "lisp",
		"js":     "node",
		"m":      "objc",
		"ml":     "ocaml",
		"pl":     "perl",
		"php":    "php",
		"py":     "python",
		"rkt":    "racket",
		"rb":     "ruby",
		"rs":     "rust",
		"scala":  "scala",
		"sh":     "bash",
	}
	return func(key string) string {
		return innerMap[key]
	}
}()

const dexecPath = "/tmp/dexec/build"
const dexecImageTemplate = "dexec/%s"
const dexecVolumeTemplate = "%s/%s:%s/%s"

// ExtractBasenameAndPermission takes an include string and splits it into
// its file or folder name and the permission string if present or the empty
// string if not.
func ExtractBasenameAndPermission(path string) (string, string) {
	pathPattern := regexp.MustCompile("([\\w.-]+)(:(rw|ro))")
	match := pathPattern.FindStringSubmatch(path)

	basename := path
	var permission string

	if len(match) == 4 {
		basename = match[1]
		permission = match[2]
	}
	return basename, permission
}

// RunDexecContainer runs an anonymouse Docker container with a Docker Exec
// image, mounting the specified sources and includes and passing the
// list of sources and arguments to the entrypoint.
func RunDexecContainer(dexecImage string, options map[OptionType][]string) {
	dockerImage := fmt.Sprintf(dexecImageTemplate, dexecImage)

	path := "."
	if len(options[TargetDir]) > 0 {
		path = options[TargetDir][0]
	}
	absPath, _ := filepath.Abs(path)

	var dockerArgs []string
	for _, source := range append(options[Source], options[Include]...) {
		basename, _ := ExtractBasenameAndPermission(source)

		dockerArgs = append(
			dockerArgs,
			[]string{
				"-v",
				fmt.Sprintf(dexecVolumeTemplate, absPath, basename, dexecPath, source),
			}...,
		)
	}

	var sourceBasenames []string
	for _, source := range options[Source] {
		basename, _ := ExtractBasenameAndPermission(source)
		sourceBasenames = append(sourceBasenames, []string{basename}...)
	}

	entrypointArgs := JoinStringSlices(
		sourceBasenames,
		AddPrefix(options[BuildArg], "-b"),
		AddPrefix(options[Arg], "-a"),
	)

	if len(options[UpdateFlag]) > 0 {
		DockerPull(dockerImage)
	}

	RunAnonymousContainer(
		dockerImage,
		dockerArgs,
		entrypointArgs,
	)
}

func validate(cli CLI) bool {
	if !IsDockerPresent() {
		log.Fatal("Docker not found")
	} else if !IsDockerRunning() {
		log.Fatal("Docker not running")
	}

	valid := false
	if len(cli.options[VersionFlag]) != 0 {
		DisplayVersion(cli.filename)
	} else if len(cli.options[Source]) == 0 ||
		len(cli.options[HelpFlag]) != 0 ||
		len(cli.options[TargetDir]) > 1 {
		DisplayHelp(cli.filename)
	} else {
		valid = true
	}
	return valid
}

func main() {
	cli := ParseOsArgs(os.Args)

	if validate(cli) {
		RunDexecContainer(
			LookupExtensionByImage(ExtractFileExtension(cli.options[Source][0])),
			cli.options,
		)
	}
}
