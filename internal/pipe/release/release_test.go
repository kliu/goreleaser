package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestPipeDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func createTmpFile(tb testing.TB, folder, path string) string {
	tb.Helper()
	f, err := os.Create(filepath.Join(folder, path))
	require.NoError(tb, err)
	require.NoError(tb, f.Close())
	return f.Name()
}

func TestRunPipeWithoutIDsThenDoesNotFilter(t *testing.T) {
	folder := t.TempDir()
	tarfile := createTmpFile(t, folder, "bin.tar.gz")
	srcfile := createTmpFile(t, folder, "source.tar.gz")
	debfile := createTmpFile(t, folder, "bin.deb")
	metafile := createTmpFile(t, folder, "metadata.json")
	checksumfile := createTmpFile(t, folder, "checksum")
	checksumsigfile := createTmpFile(t, folder, "checksum.sig")
	checksumpemfile := createTmpFile(t, folder, "checksum.pem")
	filteredtarfile := createTmpFile(t, folder, "filtered.tar.gz")
	filtereddebfile := createTmpFile(t, folder, "filtered.deb")

	config := config.Project{
		Dist: folder,
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
			IncludeMeta: true,
		},
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("v1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile,
		Extra: map[string]any{
			artifact.ExtraID: "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debfile,
		Extra: map[string]any{
			artifact.ExtraID: "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "filtered.tar.gz",
		Path: filteredtarfile,
		Extra: map[string]any{
			artifact.ExtraID: "bar",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "filtered.deb",
		Path: filtereddebfile,
		Extra: map[string]any{
			artifact.ExtraID: "bar",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableSourceArchive,
		Name: "source.tar.gz",
		Path: srcfile,
		Extra: map[string]any{
			artifact.ExtraFormat: "tar.gz",
		},
	})

	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.Checksum,
		Name: "checksum",
		Path: checksumfile,
		Extra: map[string]any{
			artifact.ExtraID: "doesnt-matter",
		},
	})

	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.Metadata,
		Name: "metadata.json",
		Path: metafile,
		Extra: map[string]any{
			artifact.ExtraID: "doesnt-matter",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.Signature,
		Name: "checksum.sig",
		Path: checksumsigfile,
		Extra: map[string]any{
			artifact.ExtraID: "bar",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.Certificate,
		Name: "checksum.pem",
		Path: checksumpemfile,
		Extra: map[string]any{
			artifact.ExtraID: "bar",
		},
	})
	client := &client.Mock{}
	require.NoError(t, doPublish(ctx, client))
	require.True(t, client.CreatedRelease)
	require.True(t, client.UploadedFile)
	require.True(t, client.ReleasePublished)
	require.Contains(t, client.UploadedFileNames, "source.tar.gz")
	require.Contains(t, client.UploadedFileNames, "bin.deb")
	require.Contains(t, client.UploadedFileNames, "bin.tar.gz")
	require.Contains(t, client.UploadedFileNames, "filtered.deb")
	require.Contains(t, client.UploadedFileNames, "filtered.tar.gz")
	require.Contains(t, client.UploadedFileNames, "metadata.json")
	require.Contains(t, client.UploadedFileNames, "checksum")
	require.Contains(t, client.UploadedFileNames, "checksum.pem")
	require.Contains(t, client.UploadedFileNames, "checksum.sig")
}

func TestRunPipeWithIDsThenFilters(t *testing.T) {
	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())
	debfile, err := os.Create(filepath.Join(folder, "bin.deb"))
	require.NoError(t, err)
	require.NoError(t, debfile.Close())
	filteredtarfile, err := os.Create(filepath.Join(folder, "filtered.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, filteredtarfile.Close())
	filtereddebfile, err := os.Create(filepath.Join(folder, "filtered.deb"))
	require.NoError(t, err)
	require.NoError(t, filtereddebfile.Close())

	config := config.Project{
		Dist: folder,
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
			IDs: []string{"foo"},
			ExtraFiles: []config.ExtraFile{
				{Glob: "./testdata/**/*"},
			},
		},
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("v1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
		Extra: map[string]any{
			artifact.ExtraID: "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debfile.Name(),
		Extra: map[string]any{
			artifact.ExtraID: "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "filtered.tar.gz",
		Path: filteredtarfile.Name(),
		Extra: map[string]any{
			artifact.ExtraID: "bar",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "filtered.deb",
		Path: filtereddebfile.Name(),
		Extra: map[string]any{
			artifact.ExtraID: "bar",
		},
	})
	client := &client.Mock{}
	require.NoError(t, doPublish(ctx, client))
	require.True(t, client.CreatedRelease)
	require.True(t, client.UploadedFile)
	require.True(t, client.ReleasePublished)
	require.Contains(t, client.UploadedFileNames, "bin.deb")
	require.Contains(t, client.UploadedFileNames, "bin.tar.gz")
	require.Contains(t, client.UploadedFileNames, "f1")
	require.NotContains(t, client.UploadedFileNames, "filtered.deb")
	require.NotContains(t, client.UploadedFileNames, "filtered.tar.gz")
}

func TestRunPipeReleaseCreationFailed(t *testing.T) {
	config := config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("v1.0.0"))
	client := &client.Mock{
		FailToCreateRelease: true,
	}
	require.Error(t, doPublish(ctx, client))
	require.False(t, client.CreatedRelease)
	require.False(t, client.UploadedFile)
	require.False(t, client.ReleasePublished)
}

func TestRunPipeWithFileThatDontExist(t *testing.T) {
	config := config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("v1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: "/nope/nope/nope",
	})
	client := &client.Mock{}
	require.Error(t, doPublish(ctx, client))
	require.True(t, client.CreatedRelease)
	require.False(t, client.UploadedFile)
}

func TestRunPipeUploadFailure(t *testing.T) {
	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())
	config := config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("v1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})
	client := &client.Mock{
		FailToUpload: true,
	}
	require.EqualError(t, doPublish(ctx, client), "failed to upload bin.tar.gz after 1 tries: upload failed")
	require.True(t, client.CreatedRelease)
	require.False(t, client.UploadedFile)
	require.False(t, client.ReleasePublished)
}

func TestRunPipeExtraFileNotFound(t *testing.T) {
	config := config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
			ExtraFiles: []config.ExtraFile{
				{Glob: "./testdata/f1.txt"},
				{Glob: "./nope"},
			},
		},
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("v1.0.0"))
	client := &client.Mock{}
	require.EqualError(t, doPublish(ctx, client), "globbing failed for pattern ./nope: matching \"./nope\": file does not exist")
	require.True(t, client.CreatedRelease)
	require.False(t, client.UploadedFile)
}

func TestRunPipeExtraOverride(t *testing.T) {
	config := config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
			ExtraFiles: []config.ExtraFile{
				{Glob: "./testdata/**/*"},
				{Glob: "./testdata/upload_same_name/f1"},
			},
		},
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("v1.0.0"))
	client := &client.Mock{}
	require.NoError(t, doPublish(ctx, client))
	require.True(t, client.CreatedRelease)
	require.True(t, client.UploadedFile)
	require.Contains(t, client.UploadedFileNames, "f1")
	require.True(t, strings.HasSuffix(client.UploadedFilePaths["f1"], "testdata/upload_same_name/f1"))
}

func TestRunPipeUploadRetry(t *testing.T) {
	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())
	config := config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	}
	ctx := testctx.NewWithCfg(config, testctx.WithCurrentTag("v1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})
	client := &client.Mock{
		FailFirstUpload: true,
	}
	require.NoError(t, doPublish(ctx, client))
	require.True(t, client.CreatedRelease)
	require.True(t, client.UploadedFile)
	require.True(t, client.ReleasePublished)
}

func TestDefault(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	ctx := testctx.NewWithCfg(
		config.Project{
			GitHubURLs: config.GitHubURLs{
				Download: "https://github.com",
			},
		},
		testctx.GitHubTokenType,
		testctx.WithCurrentTag("v1.0.0"),
	)
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Name)
	require.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Owner)
	require.Equal(t, "https://github.com/goreleaser/goreleaser/releases/tag/v1.0.0", ctx.ReleaseURL)
}

func TestDefaultInvalidURL(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser.git")
	ctx := testctx.NewWithCfg(
		config.Project{
			GitHubURLs: config.GitHubURLs{
				Download: "https://github.com",
			},
		},
		testctx.GitHubTokenType,
		testctx.WithCurrentTag("v1.0.0"),
	)
	require.Error(t, Pipe{}.Default(ctx))
}

func TestDefaultWithGitlab(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@gitlab.com:gitlabowner/gitlabrepo.git")

	ctx := testctx.NewWithCfg(
		config.Project{
			GitLabURLs: config.GitLabURLs{
				Download: "https://gitlab.com",
			},
		},
		testctx.GitLabTokenType,
		testctx.WithCurrentTag("v1.0.0"),
	)
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "gitlabrepo", ctx.Config.Release.GitLab.Name)
	require.Equal(t, "gitlabowner", ctx.Config.Release.GitLab.Owner)
	require.Equal(t, "https://gitlab.com/gitlabowner/gitlabrepo/-/releases/v1.0.0", ctx.ReleaseURL)
}

func TestDefaultWithGitlabInvalidURL(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@gitlab.com:gitlabrepo.git")
	ctx := testctx.NewWithCfg(
		config.Project{
			GitLabURLs: config.GitLabURLs{
				Download: "https://gitlab.com",
			},
		},
		testctx.GiteaTokenType,
		testctx.WithCurrentTag("v1.0.0"),
	)
	require.Error(t, Pipe{}.Default(ctx))
}

func TestDefaultWithGitea(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@gitea.example.com:giteaowner/gitearepo.git")

	ctx := testctx.NewWithCfg(
		config.Project{
			GiteaURLs: config.GiteaURLs{
				Download: "https://git.honk.com",
			},
		},
		testctx.GiteaTokenType,
		testctx.WithCurrentTag("v1.0.0"),
	)
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "gitearepo", ctx.Config.Release.Gitea.Name)
	require.Equal(t, "giteaowner", ctx.Config.Release.Gitea.Owner)
	require.Equal(t, "https://git.honk.com/giteaowner/gitearepo/releases/tag/v1.0.0", ctx.ReleaseURL)
}

func TestDefaultWithGiteaInvalidURL(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@gitea.example.com:gitearepo.git")
	ctx := testctx.NewWithCfg(
		config.Project{
			GiteaURLs: config.GiteaURLs{
				Download: "https://git.honk.com",
			},
		},
		testctx.GiteaTokenType,
		testctx.WithCurrentTag("v1.0.0"),
	)
	require.Error(t, Pipe{}.Default(ctx))
}

func TestDefaultPreRelease(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	t.Run("prerelease", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Release: config.Release{
				Prerelease: "true",
			},
		})
		ctx.TokenType = context.TokenTypeGitHub
		ctx.Semver = context.Semver{
			Major: 1,
			Minor: 0,
			Patch: 0,
		}
		require.NoError(t, Pipe{}.Default(ctx))
		require.True(t, ctx.PreRelease)
	})

	t.Run("release", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Release: config.Release{},
		})
		ctx.TokenType = context.TokenTypeGitHub
		ctx.Semver = context.Semver{
			Major:      1,
			Minor:      0,
			Patch:      0,
			Prerelease: "rc1",
		}
		require.NoError(t, Pipe{}.Default(ctx))
		require.False(t, ctx.PreRelease)
	})

	t.Run("auto-release", func(t *testing.T) {
		ctx := testctx.NewWithCfg(
			config.Project{
				Release: config.Release{
					Prerelease: "auto",
				},
			},
			testctx.GitHubTokenType,
			testctx.WithSemver(1, 0, 0, ""),
		)
		require.NoError(t, Pipe{}.Default(ctx))
		require.False(t, ctx.PreRelease)
	})

	t.Run("auto-rc", func(t *testing.T) {
		ctx := testctx.NewWithCfg(
			config.Project{
				Release: config.Release{
					Prerelease: "auto",
				},
			},
			testctx.GitHubTokenType,
			testctx.WithSemver(1, 0, 0, "rc1"),
		)
		require.NoError(t, Pipe{}.Default(ctx))
		require.True(t, ctx.PreRelease)
	})

	t.Run("auto-rc-github-setup", func(t *testing.T) {
		ctx := testctx.NewWithCfg(
			config.Project{
				Release: config.Release{
					GitHub: config.Repo{
						Name:  "foo",
						Owner: "foo",
					},
					Prerelease: "auto",
				},
			},
			testctx.GitHubTokenType,
			testctx.WithSemver(1, 0, 0, "rc1"),
		)
		require.NoError(t, Pipe{}.Default(ctx))
		require.True(t, ctx.PreRelease)
	})
}

func TestDefaultPipeDisabled(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	ctx := testctx.NewWithCfg(config.Project{
		Release: config.Release{
			Disable: "true",
		},
	})
	ctx.TokenType = context.TokenTypeGitHub
	testlib.AssertSkipped(t, Pipe{}.Default(ctx))
	require.Empty(t, ctx.Config.Release.GitHub.Name)
	require.Empty(t, ctx.Config.Release.GitHub.Owner)
}

func TestDefaultFilled(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	ctx := testctx.NewWithCfg(config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Name:  "foo",
				Owner: "bar",
			},
		},
	})
	ctx.TokenType = context.TokenTypeGitHub
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "foo", ctx.Config.Release.GitHub.Name)
	require.Equal(t, "bar", ctx.Config.Release.GitHub.Owner)
}

func TestDefaultNotAGitRepo(t *testing.T) {
	testlib.Mktmp(t)
	ctx := testctx.New(testctx.GitHubTokenType)
	require.EqualError(t, Pipe{}.Default(ctx), "current folder is not a git repository")
	require.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultGitRepoWithoutOrigin(t *testing.T) {
	testlib.Mktmp(t)
	ctx := testctx.New(testctx.GitHubTokenType)
	testlib.GitInit(t)
	require.EqualError(t, Pipe{}.Default(ctx), "no remote configured to list refs from")
	require.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultNotAGitRepoSnapshot(t *testing.T) {
	testlib.Mktmp(t)
	ctx := testctx.New(testctx.GitHubTokenType, testctx.Snapshot)
	require.NoError(t, Pipe{}.Default(ctx))
	require.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultGitRepoWithoutRemote(t *testing.T) {
	testlib.Mktmp(t)
	ctx := testctx.New(testctx.GitHubTokenType)
	require.Error(t, Pipe{}.Default(ctx))
	require.Empty(t, ctx.Config.Release.GitHub.String())
}

func TestDefaultMultipleReleasesDefined(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "githubName",
				Name:  "githubName",
			},
			GitLab: config.Repo{
				Owner: "gitlabOwner",
				Name:  "gitlabName",
			},
			Gitea: config.Repo{
				Owner: "giteaOwner",
				Name:  "giteaName",
			},
		},
	})
	require.EqualError(t, Pipe{}.Default(ctx), ErrMultipleReleases.Error())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Release: config.Release{
				Disable: "true",
			},
		})
		b, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, b)
	})

	t.Run("skip tmpl", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Env: []string{"FOO=true"},
			Release: config.Release{
				Disable: "{{ .Env.FOO }}",
			},
		})
		b, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, b)
	})

	t.Run("tmpl err", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Release: config.Release{
				Disable: "{{ .Env.FOO }}",
			},
		})
		_, err := Pipe{}.Skip(ctx)
		require.Error(t, err)
	})

	t.Run("skip upload", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Env: []string{"FOO=true"},
			Release: config.Release{
				SkipUpload: "{{ .Env.FOO }}",
			},
		})
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: "a",
			Path: "./doc.go",
			Type: artifact.UploadableFile,
		})
		client := &client.Mock{}
		testlib.AssertSkipped(t, doPublish(ctx, client))
		require.True(t, client.CreatedRelease)
		require.True(t, client.ReleasePublished)
		require.False(t, client.UploadedFile)
	})

	t.Run("skip upload tmpl", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Release: config.Release{
				SkipUpload: "true",
			},
		})
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: "a",
			Path: "./doc.go",
			Type: artifact.UploadableFile,
		})
		client := &client.Mock{}
		testlib.AssertSkipped(t, doPublish(ctx, client))
		require.True(t, client.CreatedRelease)
		require.True(t, client.ReleasePublished)
		require.False(t, client.UploadedFile)
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.New()
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: "a",
			Path: "./doc.go",
			Type: artifact.UploadableFile,
		})

		client := &client.Mock{}
		require.NoError(t, doPublish(ctx, client))
		require.True(t, client.CreatedRelease)
		require.True(t, client.ReleasePublished)
		require.True(t, client.UploadedFile)
	})
}
