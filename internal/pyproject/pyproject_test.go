package pyproject

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	proj, err := Open("./testdata/pyproject.toml")
	require.NoError(t, err)
	require.Equal(t, "python-test", proj.Project.Name)
	require.Equal(t, "0.1.0", proj.Project.Version)
	require.Equal(t, ">=3.12", proj.Project.RequiresPython)
}

func TestOpenError(t *testing.T) {
	_, err := Open("./testdata/nope.toml")
	require.Error(t, err)
}
