package main

import (
	"os"

	_ "embed"

	goversion "github.com/caarlos0/go-version"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/cmd"
	"go.uber.org/automaxprocs/maxprocs"
)

//nolint:gochecknoglobals
var (
	version   = ""
	commit    = ""
	treeState = ""
	date      = ""
	builtBy   = ""
)

func init() {
	// automatically set GOMAXPROCS to match available CPUs.
	// GOMAXPROCS will be used as the default value for the --parallelism flag.
	if _, err := maxprocs.Set(); err != nil {
		log.WithError(err).Warn("failed to set GOMAXPROCS")
	}
}

func main() {
	cmd.Execute(
		buildVersion(version, commit, date, builtBy, treeState),
		os.Exit,
		os.Args[1:],
	)
}

const website = "https://goreleaser.com"

//go:embed art.txt
var asciiArt string

func buildVersion(version, commit, date, builtBy, treeState string) goversion.Info {
	return goversion.GetVersionInfo(
		goversion.WithAppDetails("goreleaser", "Release engineering, simplified.", website),
		goversion.WithASCIIName(asciiArt),
		func(i *goversion.Info) {
			if commit != "" {
				i.GitCommit = commit
			}
			if treeState != "" {
				i.GitTreeState = treeState
			}
			if date != "" {
				i.BuildDate = date
			}
			if version != "" {
				i.GitVersion = version
			}
			if builtBy != "" {
				i.BuiltBy = builtBy
			}
		},
	)
}
