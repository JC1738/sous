package actions

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/git"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/shell"
	"github.com/samsalisbury/semv"
)

// A Server represents the `sous server` command.
type Server struct {
	Version           semv.Version
	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	Log               logging.LogSink

	ListenAddr string
	GDMRepo    string

	*config.Config
	ServerHandler http.Handler
	*sous.AutoResolver
}

// Do runs the server.
func (ss *Server) Do() error {
	if err := ensureGDMExists(ss.GDMRepo, ss.Config.StateLocation, ss.Log.Warnf); err != nil {
		return err
	}
	ss.Log.Warnf("Starting scheduled GDM resolution.")
	ss.Log.Warnf("Filtering the GDM to resolve on this server to: %v", ss.DeployFilterFlags)

	ss.AutoResolver.Kickoff()

	ss.Log.Warnf("Sous Server v%s running at %s for %s", ss.Version, ss.ListenAddr, ss.DeployFilterFlags.Cluster)

	return server.Run(ss.ListenAddr, ss.ServerHandler)
}

func ensureGDMExists(repo, localPath string, log func(string, ...interface{})) error {

	if s, err := os.Stat(localPath); err == nil && s.IsDir() {
		if files, err := ioutil.ReadDir(localPath); err != nil {
			return err
		} else if len(files) != 0 {

			// The directory exists and is not empty, do nothing.
			if repo != "" {
				log("not pulling repo %q; directory already exist and is not empty: %q", repo, localPath)
			}

			return nil
		}
	}

	if err := config.EnsureDirExists(localPath); err != nil {
		return err
	}
	// xxx Shouldn't this simply fail if there's no GDM available?
	sh, err := shell.DefaultInDir(localPath)
	if err != nil {
		return err
	}
	g, err := git.NewClient(sh)
	if err != nil {
		return err
	}
	log("cloning %q into %q ...", repo, localPath)
	if err := g.CloneRepo(repo, localPath); err != nil {
		return err
	}
	log("done")
	return nil
}
