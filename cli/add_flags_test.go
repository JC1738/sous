package cli

import (
	"bytes"
	"flag"
	"strings"
	"testing"

	"github.com/opentable/sous/config"
)

func TestAddFlagsForRectify(t *testing.T) {
	expectedHelpText := `
  -all
        all deployments should be considered
  -cluster string
        the deployment environment to consider
  -flavor string
        flavor is a short string used to differentiate alternative deployments
  -offset string
        source code relative repository offset
  -repo string
        source code repository location
`

	fs := flag.NewFlagSet("rectify", flag.ContinueOnError)

	actual := config.DeployFilterFlags{}

	if err := AddFlags(fs, &actual, RectifyFilterFlagsHelp); err != nil {
		t.Fatal(err)
	}

	expected := config.DeployFilterFlags{
		Repo:    "github.com/opentable/sous",
		Offset:  "cli",
		Cluster: "test",
		All:     true,
	}

	args := []string{
		"-repo", expected.Repo,
		"-offset", expected.Offset,
		"-cluster", expected.Cluster,
		"-all",
		// Note: this isn't really sensible, but the exclusion of all vs.
		// conditions is outside of scope
	}
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	if actual.Repo != expected.Repo {
		t.Errorf("got %q; want %q", actual.Repo, expected.Repo)
	}
	if actual.Offset != expected.Offset {
		t.Errorf("got %q; want %q", actual.Offset, expected.Offset)
	}
	if !actual.All {
		t.Errorf("got false for actual.All")
	}

	buf := &bytes.Buffer{}
	fs.SetOutput(buf)
	fs.PrintDefaults()
	actualHelp := strings.TrimSpace(buf.String())
	expectedHelp := strings.TrimSpace(expectedHelpText)
	actualFields := strings.Fields(actualHelp)
	expectedFields := strings.Fields(expectedHelp)
	// we're comparing the same words in the same order rather than being
	// concerned with whitespace differences.
	for i := range actualFields {
		if len(expectedFields)-1 < i {
			t.Errorf("got help text:\n%s\nwant:\n%s (actual longer @ %s",
				actualHelp, expectedHelp, actualFields[i])
			break
		}
		if actualFields[i] != expectedFields[i] {
			t.Errorf("got help text:\n%s\nwant:\n%s \nDiffers at word %d %s vs %s",
				actualHelp, expectedHelp, i, actualFields[i], expectedFields[i])
			break
		}
	}
}

func TestAddFlags(t *testing.T) {
	expectedHelpText := `
  -flavor string
        flavor is a short string used to differentiate alternative deployments
  -offset string
        source code relative repository offset
  -repo string
        source code repository location
  -revision string
        the ID of a revision in the repository to act upon
  -tag string
        source code revision tag
`

	fs := flag.NewFlagSet("source", flag.ContinueOnError)

	actual := config.DeployFilterFlags{}

	if err := AddFlags(fs, &actual, SourceFlagsHelp); err != nil {
		t.Fatal(err)
	}

	expected := config.DeployFilterFlags{
		Repo:     "github.com/opentable/sous",
		Offset:   "",
		Tag:      "v1.0.0",
		Revision: "cabba9e",
	}

	args := []string{
		"-repo", expected.Repo,
		"-offset", expected.Offset,
		"-tag", expected.Tag,
		"-revision", expected.Revision,
	}
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	if actual.Repo != expected.Repo {
		t.Errorf("got %q; want %q", actual.Repo, expected.Repo)
	}
	if actual.Offset != expected.Offset {
		t.Errorf("got %q; want %q", actual.Offset, expected.Offset)
	}
	if actual.Tag != expected.Tag {
		t.Errorf("got %q; want %q", actual.Tag, expected.Tag)
	}
	if actual.Revision != expected.Revision {
		t.Errorf("got %q; want %q", actual.Revision, expected.Revision)
	}
	buf := &bytes.Buffer{}
	fs.SetOutput(buf)
	fs.PrintDefaults()
	actualHelp := strings.TrimSpace(buf.String())
	expectedHelp := strings.TrimSpace(expectedHelpText)
	actualFields := strings.Fields(actualHelp)
	expectedFields := strings.Fields(expectedHelp)
	// we're comparing the same words in the same order rather than being
	// concerned with whitespace differences.
	for i := range actualFields {
		if len(expectedFields)-1 < i || (actualFields[i] != expectedFields[i]) {
			t.Errorf("got help text:\n%s\nwant:\n%s", actualHelp, expectedHelp)
		}
	}
}

func TestParseUsage(t *testing.T) {
	in := `
		-someflag
			some usage text
	`
	out, err := parseUsage(in)
	if err != nil {
		t.Fatal(err)
	}
	actual, ok := out["someflag"]
	expected := "some usage text"
	if !ok {
		t.Fatalf("no usage text for -someflag; want %q", expected)
	}
	if actual != expected {
		t.Errorf("got %q; want %q", actual, expected)
	}
}

type AddFlagsInput struct {
	FlagSet *flag.FlagSet
	Target  interface{}
	Help    string
}

type BadFlagStruct struct {
	PtrField *string
}

func newFS() *flag.FlagSet { return flag.NewFlagSet("", flag.ContinueOnError) }

func TestAddFlags_badInputs(t *testing.T) {
	var s string
	stringPtr := &s
	testError(nil, nil, "", "cannot add flags to nil *flag.FlagSet", t)
	testError(newFS(), nil, "", "target is <nil>; want pointer to struct", t)
	testError(newFS(), "", "", "target is string; want pointer to struct", t)
	testError(newFS(), config.DeployFilterFlags{}, "", "target is config.DeployFilterFlags; want pointer to struct", t)
	testError(newFS(), stringPtr, "", "target is *string; want pointer to struct", t)
	testError(newFS(), &BadFlagStruct{}, "\t-ptrfield\n\tblah", "target field cli.BadFlagStruct.PtrField is *string; want string, int, or bool", t)

	if err := AddFlags(newFS(), &config.DeployFilterFlags{}, ""); err != nil { // Not:  "no usage text for flag -repo", t)
		t.Errorf("got error %q; want no error", err)
	}

}

func testError(fs *flag.FlagSet, tgt interface{}, help string, expected string, t *testing.T) {
	t.Helper()
	in := AddFlagsInput{fs, tgt, help}
	actualErr := AddFlags(in.FlagSet, in.Target, in.Help)
	if actualErr == nil {
		t.Fatalf("got nil; want error %q", expected)
	}
	actual := actualErr.Error()
	if actual != expected {
		t.Errorf("got error %q; want error %q", actual, expected)
	}
}
