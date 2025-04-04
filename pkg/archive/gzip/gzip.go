// Package gzip implements the Archive interface providing gz archiving
// and compression.
package gzip

import (
	"fmt"
	"io"
	"os"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	gzip "github.com/klauspost/pgzip"
)

// Archive as gz.
type Archive struct {
	gw *gzip.Writer
}

// New gz archive.
func New(target io.Writer) Archive {
	// the error will be nil since the compression level is valid
	gw, _ := gzip.NewWriterLevel(target, gzip.BestCompression)
	return Archive{
		gw: gw,
	}
}

// Close all closeables.
func (a Archive) Close() error {
	return a.gw.Close()
}

// Add file to the archive.
func (a Archive) Add(f config.File) error {
	if a.gw.Name != "" {
		return fmt.Errorf("gzip: failed to add %s, only one file can be archived in gz format", f.Destination)
	}
	file, err := os.Open(f.Source) // #nosec
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	a.gw.Name = f.Destination
	if f.Info.ParsedMTime.IsZero() {
		a.gw.ModTime = info.ModTime()
	} else {
		a.gw.ModTime = f.Info.ParsedMTime
	}
	_, err = io.Copy(a.gw, file)
	return err
}
