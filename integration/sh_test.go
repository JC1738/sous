// +build commandline

package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	"github.com/opentable/sous/util/shelltest"
)

type templatedConfigs struct {
	desc.EnvDesc
	TestDir, Workdir, Homedir, Statedir string
	XDGConfig, SSHWrapper               string
	GitSSH, GitLocation, GitRemoteBase  string
	SSHExec                             string
	GoPath, ShellPath                   []string
}

func setupConfig(t *testing.T) templatedConfigs {
	descPath := os.Getenv("SOUS_QA_DESC")
	if descPath == "" {
		t.Fatalf("SOUS_QA_DESC is empty - you need to run sous_qa_setup and set that env var")
	}

	pwd := filepath.Dir(descPath)

	envDesc, err := desc.LoadDesc(descPath)
	if err != nil {
		t.Fatalf("Couldn't load a QA env description from SOUS_QA_DESC(%q): %s", descPath, err)
	}
	if !envDesc.Complete() {
		t.Fatal("Incomplete QA env description. Re-run sous_qa_setup?")
	}

	realworkdir, err := ioutil.TempDir("", "sous-cli-testing")
	if err != nil {
		t.Fatalf("Couldn't create temporary working directory: %s", err)
	}

	workdir := "/tmp/sous-work"

	_, err = os.Stat(workdir)
	if !os.IsExist(err) {
		os.Remove(workdir)
	}
	os.Symlink(realworkdir, workdir)

	sousExeDir := filepath.Join(workdir, "sous", "bin")
	sousExe := filepath.Join(sousExeDir, "sous")
	if out, err := exec.Command("go", "build", "-o", sousExe, "..").CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}

	stateDir := filepath.Join(workdir, "gdm")

	exePATH, err := shelltest.BuildPath("go", "git", "ssh", "cp", "egrep", "bash", "gcc")
	if err != nil {
		t.Fatal(err)
	}

	sshExecPath, err := exec.LookPath("ssh")
	if err != nil {
		t.Fatal(err)
	}

	testHome := filepath.Join(workdir, "home")

	gitLocation := fmt.Sprintf("%s/%d/repos", envDesc.Git.Host, envDesc.Git.Port)
	gitRemoteBase := fmt.Sprintf("ssh://root@%s/repos", envDesc.GitOrigin())
	gitSSH := envDesc.AgentIP.String()

	sshWrapper := filepath.Join(testHome, "bin/ssh_wrapper")
	firstGoPath := filepath.Join(testHome, "go")

	shellPath := []string{sousExeDir, "~/bin", exePATH, filepath.Join(firstGoPath, "bin")}

	goPath := []string{firstGoPath}

	if userGopath := os.Getenv("GOPATH"); userGopath != "" {
		for _, userGo := range strings.Split(userGopath, ":") {
			goPath = append(goPath, userGo)
			shellPath = append(shellPath, filepath.Join(userGo, "bin"))
		}
	}

	// Speculation: the size of this struct is a metric we should consider.
	return templatedConfigs{
		TestDir:       pwd,
		EnvDesc:       envDesc,
		Workdir:       workdir,
		Homedir:       testHome,
		Statedir:      stateDir,
		XDGConfig:     filepath.Join(testHome, "dot-config"),
		SSHWrapper:    sshWrapper,
		GoPath:        goPath,
		GitSSH:        gitSSH,
		SSHExec:       sshExecPath,
		GitLocation:   gitLocation,
		GitRemoteBase: gitRemoteBase,
		ShellPath:     shellPath,
	}
}

func buildShell(name string, t *testing.T) *shelltest.ShellTest {
	cfg := setupConfig(t)

	os.MkdirAll(cfg.Homedir, os.ModePerm)
	err := shelltest.TemplateConfigs(filepath.Join(cfg.TestDir, "integration/test-homedir"), cfg.Homedir, cfg)
	if err != nil {
		t.Fatalf("Templating configuration files: %+v", err)
	}

	shell := shelltest.New(t, name, cfg,
		shelltest.WithHostEnv([]string{"DOCKER_HOST", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH", "GOROOT", "CC", "CXX",
			"CC", "CONFIG_SHELL", "CXX", "NIX_BUILD_CORES", "NIX_CC", "NIX_CFLAGS_COMPILE", "NIX_ENFORCE_NO_NATIVE", "NIX_LDFLAGS", "NIX_STORE", "PYTHONPATH", "TEMP", "TEMPDIR", "_PATH", "bin", "buildInputs", "builder", "checkPhase", "goPackagePath", "outputs", "preFixup", "shell", "shellHook",
			"LD_LIBRARY_PATH", "LIBEXEC_PATH"},
			map[string]string{
				"HOME":       cfg.Homedir,
				"XDG_CONFIG": cfg.XDGConfig,
				"GIT_SSH":    cfg.SSHWrapper,
				"GOPATH":     strings.Join(cfg.GoPath, ":"),
				"PATH":       strings.Join(cfg.ShellPath, ":"),
			}))

	shell.WriteTo("raw_shell_output")

	return shell
}

func TestShellLevelIntegration(t *testing.T) {
	shell := buildShell("happypath", t)

	defaultCheck := func(name string, res shelltest.Result, t *testing.T) {
		if len(res.Errs) > 0 {
			t.Errorf("Error in %s: \n\t%s", name, res.Errs)
		}
	}

	preconditions := shell.Block("Preconditions for CLI integration tests", `
	env
	if [ -n "$GOROOT" ]; then
		mkdir -p $GOROOT
	fi
	go get github.com/nyarly/cygnus # cygnus lets us inspect Singularity for ports
	if echo "{{.EnvDesc.SingularityURL}}" | egrep -q '192.168|127.0.0'; then
		{{.TestDir}}/integration/test-registry/clean-singularity.sh {{.EnvDesc.SingularityURL}}
	fi
	cygnus -H {{.EnvDesc.SingularityURL}}
	ls {{.TestDir}}/dev_support
	`, func(name string, res shelltest.Result, t *testing.T) {
		if len(res.Errs) > 0 {
			t.Errorf("Error in %s: \n\t%s", name, res.Errs)
		}
		if res.Matches("repos>sous-server.*0_0_2") {
			msg, err := shell.Template("clean-sing", "Running sous-server already - try `./integration/test-registry/clean-singularity.sh {{.EnvDesc.SingularityURL}}`")
			if err != nil {
				t.Error("Running sous-server already - try `./integration/test-registry/clean-singularity.sh <singularity-url>`")
			}
			t.Error(msg)
		}
		if !res.Matches("sous_linux") {
			t.Error("No sous_linux available - run `make linux_build`")
		}
	})

	prologue := preconditions.Block("Test environment setup", `
	# These steps are required by the Sous integration tests
	# They're analogous to run-of-the-mill workstation maintenance.

	cd {{.TestDir}}
	env
	export SOUS_EXTRA_DOCKER_CA={{.TestDir}}/integration/test-registry/docker-registry/testing.crt
	mkdir -p {{index .GoPath 0}}/{src,bin}

	### This build gives me trouble in tests...
	### xgo does something weird and different with it's dep-cache dir
	# GOPATH={{index .GoPath 0}} make linux_build # we need Sous built for linux for the server
	go install . #install the current sous project
	cp integration/test-registry/git-server/git_pubkey_rsa* ~/dot-ssh/

	cd {{.Workdir}}
	chmod -R go-rwx ~/dot-ssh
	chmod -R +x ~/bin/*
	ssh -o ConnectTimeout=1 -o PasswordAuthentication=no -F "${HOME}/dot-ssh/config" root@{{.GitSSH}} -p 2222 /reset-repos < /dev/null
	`,
		defaultCheck)

	createGDM := prologue.Block("create the GDM", `
	git clone {{.GitRemoteBase}}/gdm
	cp ~/templated-configs/defs.yaml gdm/defs.yaml
	cat gdm/defs.yaml
	pushd gdm
	cat ~/.config/git/config >> .git/config # Eh?
	git add defs.yaml
	git commit -am "Adding defs.yaml"
	git push
	popd
	`, defaultCheck)

	// XXX There should be a `-cluster left,right` syntax, instead of two deploy commands
	setup := createGDM.Block("deploy sous server", `
	sous config
	cat ~/.config/sous/config.yaml
	git clone {{.GitRemoteBase}}/sous-server
	pushd sous-server
	export SOUS_USER_NAME=test SOUS_USER_EMAIL=test@test.com
	export SOUS_SERVER= SOUS_STATE_LOCATION={{.Statedir}}

	sous init
	sous manifest get
	sous manifest set < ~/templated-configs/sous-server.yaml
	sous manifest get # demonstrating this got to GDM

	# Last minute config
	cat Dockerfile
	cp ~/dot-ssh/git_pubkey_rsa key_sous@example.com
	cp {{.TestDir}}/dev_support/$(readlink {{.TestDir}}/dev_support/sous_linux) .
	cp {{.TestDir}}/integration/test-registry/docker-registry/testing.crt docker.crt

	ls -a
	ssh-keyscan -p 2222 {{.GitSSH}} > known_hosts

	git add key_sous@example.com known_hosts sous
	git commit -am "Adding ephemeral files"
	git tag -am "0.0.2" 0.0.2
	git push
	git push --tags

	sous build
	sous deploy -cluster left # We expect to see 'Sous is running ... in workstation mode' here:
	sous deploy -cluster right
	unset SOUS_SERVER
	unset SOUS_STATE_LOCATION
	popd
	`,
		func(name string, res shelltest.Result, t *testing.T) {
			if len(res.Errs) > 0 {
				t.Errorf("Trouble building GDM: \n\t%s", res.Errs)
			}
		})

	// This is where regular use starts
	config := setup.Block("configuration", `
	# This is kind of a hack - in normal operation, Sous would block until its
	# services had been accepted, but when bootstrapping, we need to wait for them
	# to come up.
	for n in {1..50}; do
		cygnus -H --env PORT0 {{.EnvDesc.SingularityURL}} > ~/server-singularity.txt
	  if [ $( grep sous-server ~/server-singularity.txt | wc -l) -ge 2 ]; then
		  break
		fi
	  sleep 0.1
	done
	cat ~/server-singularity.txt >2

	leftport=$(grep 'sous-server.*left' ~/server-singularity.txt | awk '{ print $3 }')
	rightport=$(grep 'sous-server.*right' ~/server-singularity.txt | awk '{ print $3 }')

	serverURL=http://{{.EnvDesc.AgentIP}}:$leftport
	echo "Determined server url as $serverURL"

	until curl -s -I $serverURL; do
	  sleep 0.1
	done
	sous config Server "$serverURL"
	echo "Set server URL to: $(sous config Server)"

	ETAG=$(curl -s -v http://192.168.99.100:$leftport/servers 2>&1 | sed -n '/Etag:/{s/.*: //; P; }')
	echo $ETAG
	sed "s/LEFTPORT/$leftport/; s/RIGHTPORT/$rightport/" < ~/templated-configs/servers.json > ~/servers.json
	cat ~/servers.json
	curl -v -X PUT -H "If-Match: ${ETAG//[$'\t\r\n ']}" -H "Content-Type: application/json" "${serverURL}/servers" --data "$(< ~/servers.json)"
	curl -s "${serverURL}/servers"
	cygnus --env TASK_HOST --env PORT0 -K -s {{.EnvDesc.SingularityURL}}
	`,
		func(name string, res shelltest.Result, t *testing.T) {
			if len(res.Errs) > 0 {
				t.Errorf("Trouble %s: \n\t%s", name, res.Errs)
			}

			if !res.Matches(`Set server URL to: http`) {
				t.Fatalf("Sous server not running!")
			}
		})

	deploy := config.Block("deploy project", `
	cd
	rm -rf sous-demo
	git clone {{.GitRemoteBase}}/sous-demo
	cd sous-demo
	git tag -am 'Release!' 0.0.23
	git push --tags
	sous init
	sous build
	sous deploy -cluster left
	`, defaultCheck)

	//check :=
	deploy.Block("confirm deployment", `
	cygnus -x 1 {{.EnvDesc.SingularityURL}}
	`, func(name string, res shelltest.Result, t *testing.T) {
		defaultCheck(name, res, t)
		if !res.Matches("sous-demo") {
			t.Error("No sous-demo request running!")
		}
	})

	update := config.Block("update project", `
	cd
	rm -rf sous-demo
	git clone {{.GitRemoteBase}}/sous-demo
	cd sous-demo
	echo "0.0.24" > customstring
	git commit -am "Updating version"
	git tag -am 'Updated!' 0.0.24
	git push --tags
	sous build
	time sous deploy -d -v -cluster left
	`, defaultCheck)

	//check :=
	update.Block("confirm update", `
	cygnus -x 1 {{.EnvDesc.SingularityURL}} > {{.Workdir}}/updated_services.txt
	cat {{.Workdir}}/updated_services.txt
	URL=$(cat {{.Workdir}}/updated_services.txt| grep "sous-demo.*left" | awk '{ print $3 ":" $4 }' | tail -n 1)
	echo $URL
	`, func(name string, res shelltest.Result, t *testing.T) {
		defaultCheck(name, res, t)
		if !res.Matches("sous-demo") {
			t.Error("No sous-demo request running!")
		}
		if !res.Matches("0_0_24") {
			t.Error("Updates sous doesn't reflect intended version")
		}
	})

	/*
		XXX This test relies on triggering an "overdue" from Singularity.
		At the moment, we hardcode a 10 minute timeout into our deploys, which is
		good for getting deploys to succeed, but makes this test more than twice as
		slow to return.

		Ideally, we'd add a GDM level configuration for timeout so that we can speed this test up.


			//failedDeploy :=
			setupOverResourced := config.Block("deploy over-resourced project", `
			cd
			rm -rf sous-demo
			git clone {{.GitRemoteBase}}/sous-demo
			cd sous-demo
			git tag -am 'Release!' 0.0.24
			git push --tags

			# We will make this deploy fail by asking for too many resources.
			sous manifest get > demo_manifest.yaml
			cat demo_manifest.yaml
			# Set CPUs to redonkulous.
			sed 's/^      cpus.*$/      cpus: "30"/g' demo_manifest.yaml > demo_manifest_toobig.yaml
			cat demo_manifest_toobig.yaml
			sous manifest set < demo_manifest_toobig.yaml
			sous build
			`, defaultCheck)

			setupOverResourced.Block("over-resourced deploy failing", `
			sous deploy -d -v -cluster right
			`, func(name string, res shelltest.Result, t *testing.T) {
				if !res.ErrsMatches(`sous deploy.*right`) {
					t.Log(res.Errs)
					t.Error("sous deploy succeeded, when it should fail!")
				}
			})
	*/

	//failedDeploy :=
	setupBadlyConfigured := config.Block("deploy badly configured project", `
	cd
	rm -rf sous-demo
	git clone {{.GitRemoteBase}}/sous-demo
	cd sous-demo
	git tag -am 'Release!' 0.0.25
	git push --tags

	# We will make this deploy fail by asking for too many resources.
	sous manifest get > demo_manifest.yaml
	cat demo_manifest.yaml
	# Set CPUs to redonkulous.
	sed 's/^      cpus.*$/      cpus: "9999999"/g' demo_manifest.yaml > demo_manifest_toobig.yaml
	cat demo_manifest_toobig.yaml
	sous manifest set < demo_manifest_toobig.yaml
	sous build
	`, defaultCheck)

	setupBadlyConfigured.Block("badly configured deploy failing", `
	sous deploy -d -v -cluster right
	`, func(name string, res shelltest.Result, t *testing.T) {
		if !res.ErrsMatches(`sous deploy.*right`) {
			t.Log(res.Errs)
			t.Error("sous deploy succeeded, when it should fail!")
		}
	})

}
