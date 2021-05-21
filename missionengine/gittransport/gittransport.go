package gittransport

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Wifi struct {
		SSID   string
		Secret string
	}
}

type GitEngine struct {
	Config           Config
	gitServerAddress string
	gitServerKey     string
	flagName         string
	fileChanges      map[string]time.Time
	filePositions    map[string]int64
}

func (me GitEngine) DataDir() string {
	return "db/" + me.flagName
}

func New(gitServerAddress string, gitServerKey string, droneName string) (*GitEngine, error) {
	flagName := time.Now().Format("20060102150405") + "-" + droneName
	err := cloneRepository(gitServerAddress, flagName)
	if err != nil {
		return nil, err
	}

	config := parseConfig(flagName)

	return &GitEngine{
		config,
		gitServerAddress,
		gitServerKey,
		flagName,
		make(map[string]time.Time),
		make(map[string]int64),
	}, nil
}

func (m *GitEngine) CommitAll() {
}

func (m *GitEngine) pullFiles() bool {
	wd, _ := os.Getwd()
	idPath := filepath.Join(wd, "/ssh/id_rsa")
	khPath := filepath.Join(wd, "/ssh/known_host_cloud")
	gitSSHCommand := fmt.Sprintf("ssh -i %s -o \"IdentitiesOnly=yes\" -o \"UserKnownHostsFile=%s\"", idPath, khPath)
	cloneCmd := exec.Command("git", "pull", "--rebase")
	cloneCmd.Env = []string{"GIT_SSH_COMMAND=" + gitSSHCommand}
	cloneCmd.Dir = "db/" + m.flagName
	cloneOut, err := cloneCmd.CombinedOutput()
	if err != nil {
		log.Printf("%s\n\nCould not pull: %v", cloneOut, err)
		return false
	}

	return true
}

func cloneRepository(gitServerAddress string, flagName string) error {
	wd, _ := os.Getwd()
	idPath := filepath.Join(wd, "/ssh/id_rsa")
	khPath := filepath.Join(wd, "/ssh/known_host_cloud")
	gitSSHCommand := fmt.Sprintf("ssh -i %s -o \"IdentitiesOnly=yes\" -o \"UserKnownHostsFile=%s\"", idPath, khPath)
	cloneCmd := exec.Command("git", "clone", gitServerAddress, "db/"+flagName)
	cloneCmd.Env = []string{"GIT_SSH_COMMAND=" + gitSSHCommand}
	cloneOut, err := cloneCmd.CombinedOutput()
	if err != nil {
		return errors.WithMessagef(err, "Could not clone: %v", cloneOut)
	}
	return nil
}

func parseConfig(flagName string) Config {
	configBytes, err := ioutil.ReadFile("db/" + flagName + "/config.yaml")
	if err != nil {
		log.Fatalf("read config: %v", err)
	}
	var configFile Config
	err = yaml.Unmarshal(configBytes, &configFile)
	if err != nil {
		log.Fatalf("unmarshal config: %v", err)
	}

	return configFile
}
