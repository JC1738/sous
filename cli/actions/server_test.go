package actions

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
)

func TestEnsureGDMExists(t *testing.T) {
	testCases := []struct {
		path       string
		createFile bool
	}{
		{"does-not-exist", false},
		{"empty-source-location", false},
		{"nonempty-source-location", true},
	}

	//Setup Test Data
	testDataDir := "testdata/gen/test-ensure-gdm-exists"

	if err := os.RemoveAll(testDataDir); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	gdmSourceRepo := path.Join(wd, testDataDir, "test-repo")

	if err := os.MkdirAll(gdmSourceRepo, 0777); err != nil {
		t.Fatal(err)
	}

	someFile := path.Join(testDataDir, "a-file")
	if err := ioutil.WriteFile(someFile, []byte("hi"), 0777); err != nil {
		t.Fatal(err)
	}

	git := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = gdmSourceRepo
		if err := c.Run(); err != nil {
			t.Fatal(err)
		}
	}

	// Create the empty source repo.
	git("init")
	git("config", "commit.gpgSign", "false")
	if err := ioutil.WriteFile(path.Join(gdmSourceRepo, "a-file"), []byte("hi"), 0777); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-m", "a message")

	//Test
	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("path: %s", testCase.path), func(t *testing.T) {
			testPath := path.Join(testDataDir, testCase.path)
			if err := os.MkdirAll(testPath, 0777); err != nil {
				t.Fatal(err)
			}

			if testCase.createFile {
				nonemptyFile := path.Join(testPath, "another-file")
				if err := ioutil.WriteFile(nonemptyFile, []byte("hi"), 0777); err != nil {
					t.Fatal(err)
				}
			}

			if err := ensureGDMExists(gdmSourceRepo, testPath, t.Logf); err != nil {
				t.Error(err)
			}
		})
	}

	//Tear Down test data
	if err := os.RemoveAll(testDataDir); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureGDMExists_notARepoFile(t *testing.T) {
	testDataDir := "testdata/gen/test-ensure-gdm-exists"

	if err := os.MkdirAll(testDataDir, 0777); err != nil {
		t.Fatal(err)
	}

	someFile := path.Join(testDataDir, "a-file")

	if err := ioutil.WriteFile(someFile, []byte("hi"), 0777); err != nil {
		t.Fatal(err)
	}

	err := ensureGDMExists(someFile, "", t.Logf)
	if err == nil {
		t.Error("expected error when GDM repo does not exist, path is valid to a file")
	}

	if err := os.RemoveAll(testDataDir); err != nil {
		t.Fatal(err)
	}

}

func TestEnsureGDMExists_notARepoDir(t *testing.T) {
	testDataDir := "testdata/gen/test-ensure-gdm-exists"

	if err := os.MkdirAll(testDataDir, 0777); err != nil {
		t.Fatal(err)
	}

	someDir := path.Join(testDataDir, "a-dir")

	if _, err := os.Stat(someDir); os.IsNotExist(err) {
		if err := os.Mkdir(someDir, 0777); err != nil {
			t.Fatal(err)
		}
	}

	err := ensureGDMExists(someDir, "", t.Logf)
	if err == nil {
		t.Error("expected error when GDM repo does not exist, path is valid to a dir")
	}

	if err := os.RemoveAll(testDataDir); err != nil {
		t.Fatal(err)
	}

}

func TestEnsureGDMExists_pathDoesNotExist(t *testing.T) {
	err := ensureGDMExists("this-path-does-not-exist", "", t.Logf)
	if err == nil {
		t.Error("expected error when GDM repo does not exist, path not valid")
	}
}
