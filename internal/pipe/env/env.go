// Package env implements the Pipe interface providing validation of
// missing environment variables needed by the release process.
package env

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	homedir "github.com/mitchellh/go-homedir"
)

// ErrMissingToken indicates an error when GITHUB_TOKEN, GITLAB_TOKEN and GITEA_TOKEN are all missing in the environment.
var ErrMissingToken = errors.New("missing GITHUB_TOKEN, GITLAB_TOKEN and GITEA_TOKEN")

// ErrMultipleTokens indicates that multiple tokens are defined. ATM only one of them if allowed.
// See https://github.com/goreleaser/goreleaser/pull/809
type ErrMultipleTokens struct {
	tokens []string
}

func (e ErrMultipleTokens) Error() string {
	return fmt.Sprintf("multiple tokens found, but only one is allowed: %s\n\nLearn more at https://goreleaser.com/errors/multiple-tokens\n", strings.Join(e.tokens, ", "))
}

// Pipe for env.
type Pipe struct{}

func (Pipe) String() string {
	return "loading environment variables"
}

func setDefaultTokenFiles(ctx *context.Context) {
	env := &ctx.Config.EnvFiles
	if env.GitHubToken == "" {
		env.GitHubToken = "~/.config/goreleaser/github_token"
	}
	if env.GitLabToken == "" {
		env.GitLabToken = "~/.config/goreleaser/gitlab_token"
	}
	if env.GiteaToken == "" {
		env.GiteaToken = "~/.config/goreleaser/gitea_token"
	}
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	templ := tmpl.New(ctx).WithEnvS(os.Environ())
	tEnv := []string{}
	for i := range ctx.Config.Env {
		env, err := templ.Apply(ctx.Config.Env[i])
		if err != nil {
			return err
		}
		tEnv = append(tEnv, env)
	}
	maps.Copy(ctx.Env, context.ToEnv(tEnv))

	setDefaultTokenFiles(ctx)
	githubToken, githubTokenErr := loadEnv("GITHUB_TOKEN", ctx.Config.EnvFiles.GitHubToken)
	gitlabToken, gitlabTokenErr := loadEnv("GITLAB_TOKEN", ctx.Config.EnvFiles.GitLabToken)
	giteaToken, giteaTokenErr := loadEnv("GITEA_TOKEN", ctx.Config.EnvFiles.GiteaToken)

	forceToken := ctx.Config.ForceToken
	if forceToken == "" {
		forceToken = os.Getenv("GORELEASER_FORCE_TOKEN")
	}
	switch strings.ToLower(forceToken) {
	case "github":
		gitlabToken = ""
		giteaToken = ""
	case "gitlab":
		githubToken = ""
		giteaToken = ""
	case "gitea":
		githubToken = ""
		gitlabToken = ""
	default:
		var tokens []string
		if githubToken != "" {
			tokens = append(tokens, "GITHUB_TOKEN")
		}
		if gitlabToken != "" {
			tokens = append(tokens, "GITLAB_TOKEN")
		}
		if giteaToken != "" {
			tokens = append(tokens, "GITEA_TOKEN")
		}
		if len(tokens) > 1 {
			return ErrMultipleTokens{tokens}
		}
	}

	noTokens := githubToken == "" && gitlabToken == "" && giteaToken == ""
	noTokenErrs := githubTokenErr == nil && gitlabTokenErr == nil && giteaTokenErr == nil

	if err := checkErrors(ctx, noTokens, noTokenErrs, gitlabTokenErr, githubTokenErr, giteaTokenErr); err != nil {
		return err
	}

	if gitlabToken != "" {
		log.Debug("token type: gitlab")
		ctx.TokenType = context.TokenTypeGitLab
		ctx.Token = gitlabToken
	}

	if giteaToken != "" {
		log.Debug("token type: gitea")
		ctx.TokenType = context.TokenTypeGitea
		ctx.Token = giteaToken
	}

	if githubToken != "" {
		log.Debug("token type: github")
		ctx.Token = githubToken
	}

	if ctx.TokenType == "" {
		ctx.TokenType = context.TokenTypeGitHub
	}

	return nil
}

func checkErrors(ctx *context.Context, noTokens, noTokenErrs bool, gitlabTokenErr, githubTokenErr, giteaTokenErr error) error {
	if ctx.SkipTokenCheck || skips.Any(ctx, skips.Publish) {
		return nil
	}
	if b, err := tmpl.New(ctx).Bool(ctx.Config.Release.Disable); err != nil || b {
		return err
	}

	if noTokens && noTokenErrs {
		return ErrMissingToken
	}

	if gitlabTokenErr != nil {
		return fmt.Errorf("failed to load gitlab token: %w", gitlabTokenErr)
	}

	if githubTokenErr != nil {
		return fmt.Errorf("failed to load github token: %w", githubTokenErr)
	}

	if giteaTokenErr != nil {
		return fmt.Errorf("failed to load gitea token: %w", giteaTokenErr)
	}
	return nil
}

func loadEnv(env, path string) (string, error) {
	val := os.Getenv(env)
	if val != "" {
		log.Infof("using token from %s", logext.Keyword("$"+env))
		return val, nil
	}
	path, err := homedir.Expand(path)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path) // #nosec
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()
	log.Infof("using token from %s", logext.Keyword(path))
	bts, _, err := bufio.NewReader(f).ReadLine()
	return string(bts), err
}
