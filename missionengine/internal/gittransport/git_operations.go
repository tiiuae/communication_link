package gittransport

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

func (m *GitTransport) pullFiles() bool {
	wd, _ := os.Getwd()
	idPath := filepath.Join(wd, "/ssh/id_rsa")
	khPath := filepath.Join(wd, "/ssh/known_host_cloud")
	gitSSHCommand := fmt.Sprintf("ssh -i %s -o \"IdentitiesOnly=yes\" -o \"UserKnownHostsFile=%s\"", idPath, khPath)
	cloneCmd := exec.Command("git", "pull", "--rebase")
	cloneCmd.Env = []string{"GIT_SSH_COMMAND=" + gitSSHCommand}
	cloneCmd.Dir = m.dataDir
	cloneOut, err := cloneCmd.CombinedOutput()
	if err != nil {
		log.Printf("%s\n\nCould not pull: %v", cloneOut, err)
		return false
	}

	return true
}

func (m *GitTransport) cloneRepository() error {
	wd, _ := os.Getwd()
	idPath := filepath.Join(wd, "/ssh/id_rsa")
	khPath := filepath.Join(wd, "/ssh/known_host_cloud")
	gitSSHCommand := fmt.Sprintf("ssh -i %s -o \"IdentitiesOnly=yes\" -o \"UserKnownHostsFile=%s\"", idPath, khPath)
	cloneCmd := exec.Command("git", "clone", m.gitServerAddress, m.dataDir)
	cloneCmd.Env = []string{"GIT_SSH_COMMAND=" + gitSSHCommand}
	cloneOut, err := cloneCmd.CombinedOutput()
	if err != nil {
		return errors.WithMessagef(err, "Could not clone: %v", cloneOut)
	}
	return nil
}
