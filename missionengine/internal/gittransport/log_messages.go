package gittransport

import (
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/tiiuae/communication_link/missionengine/internal/types"
)

func parseLogMessages(filename string, lines []string) []types.Message {
	from, to := parseFilename(filename)
	if from == "" || to == "" {
		return []types.Message{{
			Timestamp:   time.Now(),
			From:        "n/a",
			To:          "n/a",
			ID:          "n/a",
			MessageType: "n/a",
			Message:     "n/a",
		}}
	}
	messages := make([]types.Message, 0)
	for _, line := range lines {
		messages = append(messages, parseLogMessage(from, to, line))
	}

	return messages
}

func parseFilename(filename string) (string, string) {
	// [ "", "dir"; "file" ]
	splits := strings.Split(filename, "/")
	if len(splits) != 3 {
		return "", ""
	}
	dir := splits[1]
	ext := filepath.Ext(splits[2])
	file := strings.TrimSuffix(splits[2], ext)
	if strings.HasPrefix(file, "inbox") {
		fileSplit := strings.Split(file, "-")
		return fileSplit[1], dir
	}
	if strings.HasPrefix(file, "outbox") {
		return dir, "*"
	}

	return "", ""
}

func parseLogMessage(from, to, line string) types.Message {
	splits := splitLogMessage(line)
	if len(splits) != 4 {
		return errorMessage(line)
	}
	layout := "2006-01-02 15:04:05.000"
	t, err := time.Parse(layout, splits[0])
	if err != nil {
		return errorMessage(line)
	}
	return types.Message{Timestamp: t, From: from, To: to, ID: splits[1], MessageType: splits[2], Message: splits[3]}
}

func errorMessage(message string) types.Message {
	return types.Message{Timestamp: time.Now().UTC(), From: "", To: "", ID: "", MessageType: "", Message: message}
}

func splitLogMessage(line string) []string {
	var i int = 0
	f := func(c rune) bool {
		if unicode.IsSpace(c) {
			i++
			return i > 1 && i < 5
		}

		return false
	}

	return strings.FieldsFunc(line, f)
}
