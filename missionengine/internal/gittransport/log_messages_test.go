package gittransport

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/tiiuae/communication_link/missionengine/internal/types"
)

func Test_parseLogMessage(t *testing.T) {
	layout := "2006-01-02 15:04:05.000"
	ts, _ := time.Parse(layout, "2020-12-24 12:01:02.003")
	type args struct {
		line string
	}
	tests := []struct {
		name string
		args args
		want types.Message
	}{
		{"test 1", args{line: testLine(ts, "#1", "#2", "hello")}, types.Message{ts, "a", "b", "#1", "#2", "hello"}},
		{"test 2", args{line: testLine(ts, "#1", "#2", "hello world")}, types.Message{ts, "a", "b", "#1", "#2", "hello world"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseLogMessage("a", "b", tt.args.line); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testLine(ts time.Time, messageID, taskID, message string) string {
	tss := ts.Format("2006-01-02 15:04:05.000")
	return fmt.Sprintf("%s %s %s %s", tss, messageID, taskID, message)
}

func Test_parseFilename(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name  string
		args  args
		want1 string
		want2 string
	}{
		{"#1", args{""}, "", ""},
		{"#2", args{"/"}, "", ""},
		{"#3", args{"/foo.txt"}, "", ""},
		{"#4", args{"/foo.yaml"}, "", ""},
		{"#5", args{"/foo.log"}, "", ""},
		{"#6", args{"/d1/task.yaml"}, "", ""},
		{"#11", args{"/d1/inbox-d2.log"}, "d2", "d1"},
		{"#12", args{"/d1/outbox.log"}, "d1", "*"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got1, got2 := parseFilename(tt.args.filename)
			if got1 != tt.want1 {
				t.Errorf("parseFilename() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("parseFilename() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}
