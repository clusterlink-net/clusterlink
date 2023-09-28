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
	logrusStackJump          = 4
	logrusFieldlessStackJump = 6
)

func SetLog(logLevel string, logFileName string) {
	if logFileName != "" {
		usr, _ := user.Current()
		logFileFullPath := path.Join(usr.HomeDir, logFileName)
		createLogfolder(logFileFullPath)

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
	logrus.SetFormatter(&myFormatter{
		TextFormatter: &logrus.TextFormatter{
			ForceColors:     true,
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			PadLevelText:    true,
			DisableQuote:    true,
		},
	})
}

type myFormatter struct {
	*logrus.TextFormatter
}

func (f *myFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if entry.Level == logrus.ErrorLevel {
		_, file, line := f.getCurrentPosition(entry)
		entry.Data["file"] = file
		entry.Data["line"] = line
	}

	return f.TextFormatter.Format(entry)
}

func (f *myFormatter) getCurrentPosition(entry *logrus.Entry) (string, string, string) {
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

// createLogfolder create the log folder if not exists
func createLogfolder(filePath string) {
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
