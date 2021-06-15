package gittransport

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tiiuae/communication_link/missionengine/internal/types"
)

func (me *GitTransport) pushMessages(messages []types.Message) error {
	dataDir := me.dataDir
	for _, msg := range messages {
		if msg.To == "*" {
			path, file := outboxFilename(dataDir, msg.From)
			postMessage(msg.Timestamp, path, file, msg.ID, msg.MessageType, msg.Message.(string))
			log.Printf("Appending: %s/%s (%s -> %s: %s)", strings.TrimPrefix(path, dataDir), file, msg.From, msg.To, msg.MessageType)
		} else {
			path, file := inboxFilename(dataDir, msg.From, msg.To)
			postMessage(msg.Timestamp, path, file, msg.ID, msg.MessageType, msg.Message.(string))
			log.Printf("Appending: %s/%s (%s -> %s: %s)", strings.TrimPrefix(path, dataDir), file, msg.From, msg.To, msg.MessageType)
		}
	}

	// me.CommitAll()
	// me.pushFiles()

	return nil
}

func outboxFilename(dataDir, from string) (string, string) {
	return filepath.Join(dataDir, from), "outbox.log"
}

func inboxFilename(dataDir, from, to string) (string, string) {
	return filepath.Join(dataDir, to), fmt.Sprintf("inbox-%s.log", from)
}

func postMessage(timestamp time.Time, path, file, id, task, message string) (string, error) {
	fullPath := filepath.Join(path, file)
	os.Mkdir(path, os.ModeDir|os.ModePerm)
	f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer f.Close()

	ts := timestamp.Format("2006-01-02 15:04:05.000")
	content := fmt.Sprintf("%s %s %s %s\n", ts, id, task, message)
	if _, err := f.WriteString(content); err != nil {
		log.Println(err)
		return "", err
	}

	return fullPath, nil
}
