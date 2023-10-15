// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logutils

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	projectFolder            = "/.gw/"
	logrusStackJump          = 4
	logrusFieldlessStackJump = 6
)

func SetLog(logLevel string, logfile bool, logFileName string) {
	usr, _ := user.Current()
	logFileFullPath := path.Join(usr.HomeDir, projectFolder, logFileName)
	CreateProjectfolder(logFileFullPath)
	if logfile {
		f, err := os.OpenFile(logFileFullPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
		fmt.Printf("Creating log file: %v\n", logFileFullPath)
		if err != nil {
			fmt.Printf("Error opening log file: %v", err.Error())
			os.Exit(-1)
		}
		// assign it to the standard logger
		logrus.SetOutput(f)
	}
	// Set logrus
	ll, err := logrus.ParseLevel(logLevel)
	if err != nil {
		ll = logrus.ErrorLevel
	}
	logrus.SetLevel(ll)
	logrus.SetFormatter(&MyFormatter{
		TextFormatter: &logrus.TextFormatter{
			ForceColors:     true,
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			PadLevelText:    true,
			DisableQuote:    true,
		},
	})
}

type MyFormatter struct {
	*logrus.TextFormatter
}

func (f *MyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if entry.Level == logrus.ErrorLevel {
		_, file, line := f.getCurrentPosition(entry)
		entry.Data["file"] = file
		entry.Data["line"] = line
	}

	return f.TextFormatter.Format(entry)
}

func (f *MyFormatter) getCurrentPosition(entry *logrus.Entry) (string, string, string) {
	skip := logrusStackJump
	if len(entry.Data) == 0 {
		skip = logrusFieldlessStackJump
	}
start:
	pc, file, line, _ := runtime.Caller(skip)
	lineNumber := fmt.Sprintf("%d", line)

	function := runtime.FuncForPC(pc).Name()
	if strings.LastIndex(function, "sirupsen/logrus.") != -1 {
		skip++
		goto start
	}
	return function, file, lineNumber
}

func CreateProjectfolder(filePath string) {
	dir := filepath.Dir(filePath)

	// Check if the directory already exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Create the directory if it doesn't exist
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Println("Failed to create directory:", err)
			return
		}
	}
}
