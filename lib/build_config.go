package sous

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/samsalisbury/semv"
)

type (
	// BuildConfig captures the user's intent as they build a repo.
	BuildConfig struct {
		Repo, Offset, Tag, Revision string
		Strict, ForceClone          bool
		Context                     *BuildContext
	}

	// An AdvisoryName is the type for advisory tokens.
	AdvisoryName string
	// Advisories are the advisory tokens that apply to a build
	Advisories []AdvisoryName
)

const (
	// IsBuilder is an advisory that this container was used to build a finished
	// image, and should not itself be deployed.
	IsBuilder = AdvisoryName(`is a build image`)
	// UnknownRepo is an advisory that the source workspace is not a repo.
	// TODO: Disambiguate text from NoRepoAdv, they seem like the same thing.
	UnknownRepo = AdvisoryName(`source workspace lacked repo`)
	// NoRepoAdv means there is no repository.
	// TODO: Disambiguate text from UnknownRepo.
	NoRepoAdv = AdvisoryName(`no repository`)
	// NotRequestedRevision means that a different revision was built from that
	// which was requested.
	NotRequestedRevision = AdvisoryName(`requested revision not built`)
	// Unversioned means that there was no tag at the currently checked out
	// revision, or that the tag was not a semver tag, or the tag was 0.0.0.
	Unversioned = AdvisoryName(`no versioned tag`)
	// TagMismatch means that a different tag to the one which was requested was
	// built.
	TagMismatch = AdvisoryName(`tag mismatch`)
	// TagNotHead means that the requested tag exists in the history, but there
	// were more commits since, which were part of this build.
	TagNotHead = AdvisoryName(`tag not on built revision`)
	// EphemeralTag means the tag was an ephemeral tag rather than an annotated
	// tag.
	EphemeralTag = AdvisoryName(`ephemeral tag`)
	// UnpushedRev means the revision that was build is not pushed to any
	// remote.
	UnpushedRev = AdvisoryName(`unpushed revision`)
	// BogusRev means that the revision was bogus.
	// TODO: Find out what "bogus" means.
	BogusRev = AdvisoryName(`bogus revision`)
	// DirtyWS means that the workspace was dirty, which means there were
	// untracked files present, or that one or more tracked files were modified
	// since the last commit.
	DirtyWS = AdvisoryName(`dirty workspace`)
	// RevisionRebuilt means that there is already an artifact built for this revision/tag pair.
	RevisionRebuilt = AdvisoryName(`tag,revisions already built and registered`)
)

func (ads Advisories) merge(other Advisories) Advisories {
	child := Advisories{}
	child = append(child, ads...)

Item:
	for _, a := range other {
		for _, z := range child {
			if z == a {
				continue Item
			}
		}
		child = append(child, a)
	}
	return child
}

func (ads Advisories) strings() []string {
	strs := []string{}
	for _, a := range ads {
		strs = append(strs, string(a))
	}
	return strs
}

// NewContext returns a new BuildContext updated based on the user's intent as expressed in the Config
func (c *BuildConfig) NewContext() *BuildContext {
	ctx := c.Context
	sc := c.Context.Source
	sh := ctx.Sh.Clone()
	sh.CD(sc.RootDir)
	bc := BuildContext{
		Sh:      sh,
		Scratch: ctx.Scratch,
		Machine: ctx.Machine,
		User:    ctx.User,
		Changes: ctx.Changes,
		Source: SourceContext{
			OffsetDir:      c.chooseOffset(),
			RemoteURL:      c.chooseRemoteURL(),
			NearestTagName: c.chooseTag(),

			RootDir:            sc.RootDir,
			Branch:             sc.Branch,
			Revision:           sc.Revision,
			Files:              sc.Files,
			ModifiedFiles:      sc.ModifiedFiles,
			NewFiles:           sc.NewFiles,
			Tags:               sc.Tags,
			NearestTagRevision: sc.NearestTagRevision,
			PrimaryRemoteURL:   sc.PrimaryRemoteURL,
			RemoteURLs:         sc.RemoteURLs,
			DirtyWorkingTree:   sc.DirtyWorkingTree,
			RevisionUnpushed:   sc.RevisionUnpushed,
		},
	}

	bc.Advisories = c.Advisories(&bc)

	return &bc
}

func (c *BuildConfig) chooseRemoteURL() string {
	if c.Repo == "" {
		Log.Debug.Printf("Using best guess: % #v", c.Context.Source.PrimaryRemoteURL)
		return c.Context.Source.PrimaryRemoteURL
	}
	return c.Repo
}

func (c *BuildConfig) chooseTag() string {
	if c.Tag == "" {
		return c.Context.Source.NearestTagName
	}
	return c.Tag
}

func (c *BuildConfig) chooseOffset() string {
	if c.Offset == "" {
		return c.Context.Source.OffsetDir
	}
	clean := filepath.Clean(c.Offset)
	if clean == "." {
		return ""
	}
	return clean
}

// Resolve settles configurations so that e.g. captured version tags are used in the absence of user input
func (c *BuildConfig) Resolve() {
	c.Tag = c.chooseTag()
}

// Validate checks that the Config is well formed
func (c *BuildConfig) Validate() error {
	if _, ve := semv.Parse(c.Tag); ve != nil {
		return fmt.Errorf("semver git tag required.  invalid tag: %q", c.Tag)
	}
	return nil
}

// GuardStrict returns an error if there are imperfections in the proposed build
func (c *BuildConfig) GuardStrict(bc *BuildContext) error {
	if !c.Strict {
		return nil
	}
	as := bc.Advisories
	if len(as) > 0 {
		return fmt.Errorf("Strict built encountered advisories:\n  %s", strings.Join(as.strings(), "  \n"))
	}
	return nil
}

// GuardRegister returns an error if any development-only advisories exist
func (c *BuildConfig) GuardRegister(bc *BuildContext) error {
	var blockers Advisories
	for _, a := range bc.Advisories {
		switch AdvisoryName(a) {
		case DirtyWS, UnpushedRev, NoRepoAdv, NotRequestedRevision:
			blockers = append(blockers, a)
		}
	}
	if len(blockers) > 0 {
		return fmt.Errorf("build may not be deployable in all clusters due to advisories:\n  %s", strings.Join(blockers.strings(), "\n  "))
	}
	return nil
}

// Advisories returns a list of advisories that apply to ctx.
func (c *BuildConfig) Advisories(ctx *BuildContext) Advisories {
	advs := Advisories{}
	s := ctx.Source
	knowsRepo := false
	for _, r := range s.RemoteURLs {
		if s.RemoteURL == r {
			knowsRepo = true
			break
		}
	}
	if !knowsRepo {
		advs = append(advs, UnknownRepo)
	}

	if s.RemoteURL == "" {
		advs = append(advs, NoRepoAdv)
	}

	if c.Revision != "" && c.Revision != s.Revision {
		advs = append(advs, NotRequestedRevision)
	}

	if c.Context.Source.Version().Version.Format(`M.m.p`) == `0.0.0` {
		advs = append(advs, Unversioned)
	}

	if c.Tag != "" {
		hasTag := false
		for _, t := range s.Tags {
			if t.Name == c.Tag {
				hasTag = true
				break
			}
		}
		if !hasTag {
			advs = append(advs, EphemeralTag)
		} else if s.NearestTagRevision != s.Revision {
			Log.Debug.Printf("%s != %s", s.NearestTagRevision, s.Revision)
			advs = append(advs, (TagNotHead))
		}
	}

	if s.DirtyWorkingTree {
		advs = append(advs, DirtyWS)
	}

	if s.RevisionUnpushed {
		advs = append(advs, UnpushedRev)
	}

	return advs
}
