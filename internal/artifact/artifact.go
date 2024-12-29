// Package artifact provides the core artifact storage for goreleaser.
package artifact

//nolint:gosec
import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/experimental"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/sha3"
)

// Type defines the type of an artifact.
type Type int

// If you add more types, update TestArtifactTypeStringer!
const (
	// UploadableArchive a tar.gz/zip archive to be uploaded.
	UploadableArchive Type = iota + 1
	// UploadableBinary is a binary file to be uploaded.
	UploadableBinary
	// UploadableFile is any file that can be uploaded.
	UploadableFile
	// Binary is a binary (output of a gobuild).
	Binary
	// UniversalBinary is a binary that contains multiple binaries within.
	UniversalBinary
	// LinuxPackage is a linux package generated by nfpm.
	LinuxPackage
	// PublishableSnapcraft is a snap package yet to be published.
	PublishableSnapcraft
	// Snapcraft is a published snap package.
	Snapcraft
	// PublishableDockerImage is a Docker image yet to be published.
	PublishableDockerImage
	// DockerImage is a published Docker image.
	DockerImage
	// DockerManifest is a published Docker manifest.
	DockerManifest
	// Checksum is a checksums file.
	Checksum
	// Signature is a signature file.
	Signature
	// Certificate is a signing certificate file
	Certificate
	// UploadableSourceArchive is the archive with the current commit source code.
	UploadableSourceArchive
	// BrewTap is an uploadable homebrew tap recipe file.
	BrewTap
	// Nixpkg is an uploadable nix package.
	Nixpkg
	// WingetInstaller winget installer file.
	WingetInstaller
	// WingetDefaultLocale winget default locale file.
	WingetDefaultLocale
	// WingetVersion winget version file.
	WingetVersion
	// PkgBuild is an Arch Linux AUR PKGBUILD file.
	PkgBuild
	// SrcInfo is an Arch Linux AUR .SRCINFO file.
	SrcInfo
	// KrewPluginManifest is a krew plugin manifest file.
	KrewPluginManifest
	// ScoopManifest is an uploadable scoop manifest file.
	ScoopManifest
	// SBOM is a Software Bill of Materials file.
	SBOM
	// PublishableChocolatey is a chocolatey package yet to be published.
	PublishableChocolatey
	// Header is a C header file, generated for CGo library builds.
	Header
	// CArchive is a C static library, generated via a CGo build with buildmode=c-archive.
	CArchive
	// CShared is a C shared library, generated via a CGo build with buildmode=c-shared.
	CShared
	// Metadata is an internal goreleaser metadata JSON file.
	Metadata
)

func (t Type) String() string {
	switch t {
	case UploadableArchive:
		return "Archive"
	case UploadableFile:
		return "File"
	case UploadableBinary, Binary, UniversalBinary:
		return "Binary"
	case LinuxPackage:
		return "Linux Package"
	case PublishableDockerImage:
		return "Docker Image"
	case DockerImage:
		return "Published Docker Image"
	case DockerManifest:
		return "Docker Manifest"
	case PublishableSnapcraft, Snapcraft:
		return "Snap"
	case Checksum:
		return "Checksum"
	case Signature:
		return "Signature"
	case Certificate:
		return "Certificate"
	case UploadableSourceArchive:
		return "Source"
	case BrewTap:
		return "Brew Tap"
	case KrewPluginManifest:
		return "Krew Plugin Manifest"
	case ScoopManifest:
		return "Scoop Manifest"
	case SBOM:
		return "SBOM"
	case PkgBuild:
		return "PKGBUILD"
	case SrcInfo:
		return "SRCINFO"
	case PublishableChocolatey:
		return "Chocolatey"
	case Header:
		return "C Header"
	case CArchive:
		return "C Archive Library"
	case CShared:
		return "C Shared Library"
	case WingetInstaller, WingetDefaultLocale, WingetVersion:
		return "Winget Manifest"
	case Nixpkg:
		return "Nixpkg"
	case Metadata:
		return "Metadata"
	default:
		return "unknown"
	}
}

const (
	ExtraID         = "ID"
	ExtraBinary     = "Binary"
	ExtraExt        = "Ext" // should always have the preceding '.'
	ExtraFormat     = "Format"
	ExtraWrappedIn  = "WrappedIn"
	ExtraBinaries   = "Binaries"
	ExtraRefresh    = "Refresh"
	ExtraReplaces   = "Replaces"
	ExtraDigest     = "Digest"
	ExtraSize       = "Size"
	ExtraChecksum   = "Checksum"
	ExtraChecksumOf = "ChecksumOf"
	ExtraBuilder    = "Builder"
)

// Extras represents the extra fields in an artifact.
type Extras map[string]any

func (e Extras) MarshalJSON() ([]byte, error) {
	m := map[string]any{}
	for k, v := range e {
		if k == ExtraRefresh {
			// refresh is a func, so we can't serialize it.
			continue
		}
		m[k] = v
	}
	return json.Marshal(m)
}

// Artifact represents an artifact and its relevant info.
type Artifact struct {
	Name      string `json:"name,omitempty"`
	Path      string `json:"path,omitempty"`
	Goos      string `json:"goos,omitempty"`
	Goarch    string `json:"goarch,omitempty"`
	Goamd64   string `json:"goamd64,omitempty"`
	Go386     string `json:"go386,omitempty"`
	Goarm     string `json:"goarm,omitempty"`
	Goarm64   string `json:"goarm64,omitempty"`
	Gomips    string `json:"gomips,omitempty"`
	Goppc64   string `json:"goppc64,omitempty"`
	Goriscv64 string `json:"goriscv64,omitempty"`
	Target    string `json:"target,omitempty"`
	Type      Type   `json:"internal_type,omitempty"`
	TypeS     string `json:"type,omitempty"`
	Extra     Extras `json:"extra,omitempty"`
}

func (a Artifact) String() string {
	return a.Name
}

// Extra tries to get the extra field with the given name, returning either
// its value, the default value for its type, or an error.
//
// If the extra value cannot be cast into the given type, it'll try to convert
// it to JSON and unmarshal it into the correct type after.
//
// If that fails as well, it'll error.
func Extra[T any](a Artifact, key string) (T, error) {
	ex := a.Extra[key]
	if ex == nil {
		return *(new(T)), nil
	}

	t, ok := ex.(T)
	if ok {
		return t, nil
	}

	bts, err := json.Marshal(ex)
	if err != nil {
		return t, err
	}

	decoder := json.NewDecoder(bytes.NewReader(bts))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&t)
	return t, err
}

// ExtraOr returns the Extra field with the given key or the or value specified
// if it is nil.
func ExtraOr[T any](a Artifact, key string, or T) T {
	if a.Extra[key] == nil {
		return or
	}
	return a.Extra[key].(T)
}

// Checksum calculates the checksum of the artifact.
//
//nolint:gosec
func (a Artifact) Checksum(algorithm string) (string, error) {
	log.Debugf("calculating checksum for %s", a.Path)
	file, err := os.Open(a.Path)
	if err != nil {
		return "", fmt.Errorf("failed to checksum: %w", err)
	}
	defer file.Close()
	var h hash.Hash
	switch algorithm {
	case "blake2b":
		h, err = blake2b.New512(nil)
		if err != nil {
			return "", fmt.Errorf("failed to checksum: %w", err)
		}
	case "blake2s":
		h, err = blake2s.New256(nil)
		if err != nil {
			return "", fmt.Errorf("failed to checksum: %w", err)
		}
	case "crc32":
		h = crc32.NewIEEE()
	case "md5":
		h = md5.New()
	case "sha224":
		h = sha256.New224()
	case "sha384":
		h = sha512.New384()
	case "sha256":
		h = sha256.New()
	case "sha1":
		h = sha1.New()
	case "sha512":
		h = sha512.New()
	case "sha3-224":
		h = sha3.New224()
	case "sha3-384":
		h = sha3.New384()
	case "sha3-256":
		h = sha3.New256()
	case "sha3-512":
		h = sha3.New512()
	default:
		return "", fmt.Errorf("invalid algorithm: %s", algorithm)
	}

	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("failed to checksum: %w", err)
	}
	check := hex.EncodeToString(h.Sum(nil))
	if a.Extra == nil {
		a.Extra = make(Extras)
	}
	a.Extra[ExtraChecksum] = fmt.Sprintf("%s:%s", algorithm, check)
	return check, nil
}

var noRefresh = func() error { return nil }

// Refresh executes a Refresh extra function on artifacts, if it exists.
func (a Artifact) Refresh() error {
	// for now lets only do it for checksums, as we know for a fact that
	// they are the only ones that support this right now.
	if a.Type != Checksum {
		return nil
	}
	if err := ExtraOr(a, ExtraRefresh, noRefresh)(); err != nil {
		return fmt.Errorf("failed to refresh %q: %w", a.Name, err)
	}
	return nil
}

// ID returns the artifact ID if it exists, empty otherwise.
func (a Artifact) ID() string {
	return ExtraOr(a, ExtraID, "")
}

// Format returns the artifact Format if it exists, empty otherwise.
func (a Artifact) Format() string {
	return ExtraOr(a, ExtraFormat, "")
}

// Artifacts is a list of artifacts.
type Artifacts struct {
	items []*Artifact
	lock  *sync.Mutex
}

// New return a new list of artifacts.
func New() *Artifacts {
	return &Artifacts{
		items: []*Artifact{},
		lock:  &sync.Mutex{},
	}
}

// Refresh visits all artifacts and refreshes them.
func (artifacts *Artifacts) Refresh() error {
	return artifacts.Visit(func(a *Artifact) error {
		return a.Refresh()
	})
}

// List return the actual list of artifacts.
func (artifacts *Artifacts) List() []*Artifact {
	artifacts.lock.Lock()
	defer artifacts.lock.Unlock()
	return artifacts.items
}

// GroupByID groups the artifacts by their ID.
func (artifacts *Artifacts) GroupByID() map[string][]*Artifact {
	result := map[string][]*Artifact{}
	for _, a := range artifacts.List() {
		id := a.ID()
		if id == "" {
			continue
		}
		result[a.ID()] = append(result[a.ID()], a)
	}
	return result
}

// GroupByPlatform groups the artifacts by their platform.
func (artifacts *Artifacts) GroupByPlatform() map[string][]*Artifact {
	// we'll try to keep the most basic platform as group (goos+goarch).
	// we'll though group it further if we have multiple goarm, goamd64, or
	// gomips, to keep compatibility with previous versions of goreleaser.
	simpleResult := map[string][]*Artifact{}
	specificResult := map[string][]*Artifact{}
	goamd64s := map[string]struct{}{}
	gomipses := map[string]struct{}{}
	goarms := map[string]struct{}{}
	for _, a := range artifacts.List() {
		plat := a.Goos + a.Goarch
		fullplat := plat + a.Goarm + a.Gomips + a.Goamd64
		goamd64s[a.Goamd64] = struct{}{}
		gomipses[a.Gomips] = struct{}{}
		goarms[a.Goarm] = struct{}{}
		simpleResult[plat] = append(simpleResult[plat], a)
		specificResult[fullplat] = append(specificResult[fullplat], a)
	}

	if len(nonEmpty(goamd64s)) > 1 ||
		len(nonEmpty(gomipses)) > 1 ||
		len(nonEmpty(goarms)) > 1 {
		return specificResult
	}

	return simpleResult
}

func nonEmpty(m map[string]struct{}) []string {
	return slices.DeleteFunc(
		slices.Collect(maps.Keys(m)),
		func(s string) bool {
			return s == ""
		},
	)
}

func relPath(a *Artifact) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(a.Path, cwd) {
		return "", nil
	}
	return filepath.Rel(cwd, a.Path)
}

func shouldRelPath(a *Artifact) bool {
	switch a.Type {
	case DockerImage, DockerManifest, PublishableDockerImage:
		return false
	default:
		return filepath.IsAbs(a.Path)
	}
}

// Add safely adds a new artifact to an artifact list.
func (artifacts *Artifacts) Add(a *Artifact) {
	artifacts.lock.Lock()
	defer artifacts.lock.Unlock()
	a.Name = cleanName(*a)
	if shouldRelPath(a) {
		rel, err := relPath(a)
		if rel != "" && err == nil {
			a.Path = rel
		}
	}
	a.Path = filepath.ToSlash(a.Path)
	log.WithField("name", a.Name).
		WithField("type", a.Type).
		WithField("path", a.Path).
		Debug("added new artifact")
	artifacts.items = append(artifacts.items, a)
}

// Remove removes artifacts that match the given filter from the original artifact list.
func (artifacts *Artifacts) Remove(filter Filter) error {
	if filter == nil {
		return nil
	}

	artifacts.lock.Lock()
	defer artifacts.lock.Unlock()

	result := New()
	for _, a := range artifacts.items {
		if filter(a) {
			log.WithField("name", a.Name).
				WithField("type", a.Type).
				WithField("path", a.Path).
				Debug("removing")
		} else {
			result.items = append(result.items, a)
		}
	}

	artifacts.items = result.items
	return nil
}

// Filter defines an artifact filter which can be used within the Filter
// function.
type Filter func(a *Artifact) bool

// OnlyReplacingUnibins removes universal binaries that did not replace the single-arch ones.
//
// This is useful specially on homebrew et al, where you'll want to use only either the single-arch or the universal binaries.
func OnlyReplacingUnibins(a *Artifact) bool {
	return ExtraOr(*a, ExtraReplaces, true)
}

// ByGoos is a predefined filter that filters by the given goos.
func ByGoos(s string) Filter {
	return func(a *Artifact) bool {
		return a.Goos == s
	}
}

// ByGoarch is a predefined filter that filters by the given goarch.
func ByGoarch(s string) Filter {
	return func(a *Artifact) bool {
		return a.Goarch == s
	}
}

// ByGoarm is a predefined filter that filters by the given goarm.
func ByGoarm(s string) Filter {
	return func(a *Artifact) bool {
		switch ExtraOr(*a, ExtraBuilder, "") {
		case "zig", "rust":
			return s == experimental.DefaultGOARM()
		default:
			return a.Goarm == s
		}
	}
}

// ByGoamd64 is a predefined filter that filters by the given goamd64.
func ByGoamd64(s string) Filter {
	return func(a *Artifact) bool {
		switch ExtraOr(*a, ExtraBuilder, "") {
		case "zig", "rust":
			return s == "v1"
		default:
			return a.Goamd64 == s
		}
	}
}

// ByType is a predefined filter that filters by the given type.
func ByType(t Type) Filter {
	return func(a *Artifact) bool {
		return a.Type == t
	}
}

// ByFormats filters artifacts by a `Format` extra field.
func ByFormats(formats ...string) Filter {
	filters := make([]Filter, 0, len(formats))
	for _, format := range formats {
		filters = append(filters, func(a *Artifact) bool {
			return a.Format() == format
		})
	}
	return Or(filters...)
}

// ByIDs filter artifacts by an `ID` extra field.
func ByIDs(ids ...string) Filter {
	filters := make([]Filter, 0, len(ids))
	for _, id := range ids {
		filters = append(filters, func(a *Artifact) bool {
			// checksum and source archive are always for all artifacts, so return always true.
			return a.Type == Checksum ||
				a.Type == UploadableSourceArchive ||
				a.Type == UploadableFile ||
				a.Type == Metadata ||
				a.ID() == id
		})
	}
	return Or(filters...)
}

// ByExt filter artifact by their 'Ext' extra field.
//
// The comp is done ignoring the preceding '.', so `ByExt("deb")` and
// `ByExt(".deb")` have the same result.
func ByExt(exts ...string) Filter {
	filters := make([]Filter, 0, len(exts))
	for _, ext := range exts {
		filters = append(filters, func(a *Artifact) bool {
			actual := ExtraOr(*a, ExtraExt, "")
			return strings.TrimPrefix(actual, ".") == strings.TrimPrefix(ext, ".")
		})
	}
	return Or(filters...)
}

// ByBinaryLikeArtifacts filter artifacts down to artifacts that are Binary, UploadableBinary, or UniversalBinary,
// deduplicating artifacts by path (preferring UploadableBinary over all others). Note: this filter is unique in the
// sense that it cannot act in isolation of the state of other artifacts; the filter requires the whole list of
// artifacts in advance to perform deduplication.
func ByBinaryLikeArtifacts(arts *Artifacts) Filter {
	// find all of the paths for any uploadable binary artifacts
	uploadableBins := arts.Filter(ByType(UploadableBinary)).List()
	uploadableBinPaths := map[string]struct{}{}
	for _, a := range uploadableBins {
		uploadableBinPaths[a.Path] = struct{}{}
	}

	// we want to keep any matching artifact that is not a binary that already has a path accounted for
	// by another uploadable binary. We always prefer uploadable binary artifacts over binary artifacts.
	deduplicateByPath := func(a *Artifact) bool {
		if a.Type == UploadableBinary {
			return true
		}
		_, ok := uploadableBinPaths[a.Path]
		return !ok
	}

	return And(
		// allow all of the binary-like artifacts as possible...
		Or(
			ByType(Binary),
			ByType(UploadableBinary),
			ByType(UniversalBinary),
		),
		// ... but remove any duplicates found
		deduplicateByPath,
	)
}

// Or performs an OR between all given filters.
func Or(filters ...Filter) Filter {
	return func(a *Artifact) bool {
		for _, f := range filters {
			if f(a) {
				return true
			}
		}
		return false
	}
}

// And performs an AND between all given filters.
func And(filters ...Filter) Filter {
	return func(a *Artifact) bool {
		for _, f := range filters {
			if !f(a) {
				return false
			}
		}
		return true
	}
}

// Filter filters the artifact list, returning a new instance.
// There are some pre-defined filters but anything of the Type Filter
// is accepted.
// You can compose filters by using the And and Or filters.
func (artifacts *Artifacts) Filter(filter Filter) *Artifacts {
	if filter == nil {
		return artifacts
	}

	result := New()
	for _, a := range artifacts.List() {
		if filter(a) {
			result.items = append(result.items, a)
		}
	}
	return result
}

// Paths returns the artifact.Path of the current artifact list.
func (artifacts *Artifacts) Paths() []string {
	var result []string
	for _, artifact := range artifacts.List() {
		result = append(result, artifact.Path)
	}
	return result
}

// VisitFn is a function that can be executed against each artifact in a list.
type VisitFn func(a *Artifact) error

// Visit executes the given function for each artifact in the list.
func (artifacts *Artifacts) Visit(fn VisitFn) error {
	for _, artifact := range artifacts.List() {
		if err := fn(artifact); err != nil {
			return err
		}
	}
	return nil
}

func cleanName(a Artifact) string {
	name := a.Name
	ext := filepath.Ext(name)
	result := strings.TrimSpace(strings.TrimSuffix(name, ext)) + ext
	if name != result {
		log.WithField("name", a.Name).
			WithField("new name", result).
			WithField("type", a.Type).
			WithField("path", a.Path).
			Warn("removed trailing whitespaces from artifact name")
	}
	return result
}
