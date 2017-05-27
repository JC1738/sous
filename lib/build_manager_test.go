package sous

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func rootedBuildManager(root, offset string) *BuildManager {
	return &BuildManager{
		BuildConfig: &BuildConfig{
			Context: &BuildContext{
				Source: SourceContext{
					RootDir:   root,
					OffsetDir: offset,
				},
			},
		},
	}
}

func TestOffsetFromWorkdir(t *testing.T) {
	root := "/somewhere/project"
	relWork := "working"
	cliArg := "offset"
	off := "working/offset"
	bm := rootedBuildManager(root, relWork)

	err := bm.OffsetFromWorkdir(cliArg)
	if err != nil {
		t.Errorf("error not nil: %v", err)
	}
	if bm.BuildConfig.Offset != off {
		t.Errorf("%q != %q", bm.BuildConfig.Offset, off)
	}
}

func TestOffsetFromWorkdir_OSXTmpDir(t *testing.T) {
	base, err := ioutil.TempDir("", "osxtmpdir")
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(base, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	base, err = filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}

	root := filepath.Join(base, "root/dir/here")
	err = os.MkdirAll(root, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	pwd := filepath.Join(base, "working")
	err = os.Symlink(root, pwd)
	if err != nil {
		t.Fatal(err)
	}

	if pwd == root {
		t.Fatal("The test assumption that these would be different directories was flawed")
	}

	off := ""

	offset, err := NormalizedOffset(root, pwd)
	if err != nil {
		t.Fatal(err)
	}
	bm := rootedBuildManager(root, offset)
	err = bm.OffsetFromWorkdir(".")

	if err != nil {
		t.Errorf("error not nil: %v", err)
	}
	if bm.BuildConfig.Offset != off {
		t.Errorf("%q != %q", bm.BuildConfig.Offset, off)
	}
}

func TestOffsetFromWorkdirOffsetOutsideOfRoot(t *testing.T) {
	root := "/somewhere/project"
	cliArg := ".."
	bm := rootedBuildManager(root, "")

	err := bm.OffsetFromWorkdir(cliArg)
	if err == nil {
		t.Errorf("error nil when offset outside of project root")
	}
	if bm.BuildConfig.Offset != "" {
		t.Errorf("Offset = %q", bm.BuildConfig.Offset)
	}
}

func TestOffsetFromWorkdirWhenOffsetAlreadyConfigd(t *testing.T) {
	root := "/somewhere/project"
	relWork := "working"
	cliArg := "offset"
	off := "working/offset"
	flagOffset := "working/elsewhere"

	bm := rootedBuildManager(root, relWork)
	bm.BuildConfig.Offset = flagOffset
	err := bm.OffsetFromWorkdir(cliArg)

	if err == nil {
		t.Errorf("error nil when offset already set")
	}
	if bm.BuildConfig.Offset != flagOffset {
		t.Errorf("%q = %q (not %q)", bm.BuildConfig.Offset, off, flagOffset)
	}
}

type FakeRegistrar struct{}

func (FakeRegistrar) Register(*BuildResult, *BuildContext) error { return nil }

func TestBuildManager_RegisterAndWarnAdvisories_withAdvisories(t *testing.T) {
	br := &BuildResult{}
	bc := &BuildContext{
		Advisories: Advisories{DirtyWS},
	}
	m := &BuildManager{
		Registrar: FakeRegistrar{},
	}
	if err := m.RegisterAndWarnAdvisories(br, bc); err != nil {
		t.Fatal(err)
	}
}

func TestBuildManager_RegisterAndWarnAdvisories_noAdvisories(t *testing.T) {
	br := &BuildResult{}
	bc := &BuildContext{
		Advisories: Advisories{},
	}
	m := &BuildManager{
		Registrar: FakeRegistrar{},
	}
	if err := m.RegisterAndWarnAdvisories(br, bc); err != nil {
		t.Fatal(err)
	}
}
