package git

import (
	"os/exec"
	"strings"
	"errors"
)

func CurrentTag() (tag string, err error) {
	return getTag("master")
}

func PreviousTag(base string) (tag string, err error) {
	return getTag(base + "^")
}

func getTag(ref string) (tag string, err error) {
	cmd := exec.Command(
		"git",
		"describe",
		"--tags",
		"--abbrev=0",
		"--always",
		ref,
	)
	bts, err := cmd.CombinedOutput()
	if err != nil {
		return tag, errors.New(err.Error() + ": " + string(bts))
	}
	return strings.Split(string(bts), "\n")[0], err
}
