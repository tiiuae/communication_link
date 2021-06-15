package gittransport

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tiiuae/communication_link/missionengine/internal/types"
)

func (me *GitTransport) pullMessages() []types.Message {
	droneName := me.deviceID
	newCommits := me.pullFiles()
	messages := make([]types.Message, 0)
	if newCommits {
		filesChanged := me.diffFiles()
		for _, absPath := range filesChanged {
			relPath := strings.TrimPrefix(absPath, me.dataDir)
			from, to := parseFilename(relPath)
			if (to == droneName || to == "*") && from != droneName {
				log.Printf("Parsing log file: %s", relPath)
				lines, _ := me.readFile(absPath)
				msgs := parseLogMessages(relPath, lines)
				messages = append(messages, msgs...)
			} else {
				log.Printf("Skipping file: %s (%s -> %s)", relPath, from, to)
			}
		}
	}

	return messages
}

func (me *GitTransport) diffFiles() []string {
	dataDir := me.dataDir
	cache := me.fileChanges
	results := make([]string, 0)
	filepath.Walk(dataDir, func(root string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		filename := root
		modTime := info.ModTime()
		cached, ok := cache[filename]

		if !ok {
			cache[filename] = modTime
			results = append(results, filename)
		} else if cached.Before(modTime) {
			cache[filename] = modTime
			results = append(results, filename)
		}

		return nil
	})

	return results
}

func (me *GitTransport) readFile(filename string) ([]string, error) {
	pos := me.filePositions[filename]
	lines, newPos, err := readLinesFrom(filename, pos)
	me.filePositions[filename] = newPos

	return lines, err
}

func readLinesFrom(filename string, pos int64) ([]string, int64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, nil
	}

	_, err = file.Seek(pos, os.SEEK_SET)
	if err != nil {
		return nil, 0, nil
	}

	r := bufio.NewReader(file)

	lines := make([]string, 0)
	var line string

	for i := 1; ; i++ {
		line, err = r.ReadString('\n')

		if err != nil {
			break
		}
		pos += int64(len(line))
		lines = append(lines, line)
	}

	if err != nil && err != io.EOF {
		return nil, 0, nil
	}

	return lines, pos, nil
}
