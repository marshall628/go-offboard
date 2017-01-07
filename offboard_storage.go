package storage

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	jww "github.com/spf13/jwalterweatherman"
)

// initializeLogging: jwalterweatherman logging package
func initializeLogging(verbose bool, logfile string) {
	file, err := os.OpenFile(logfile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		jww.CRITICAL.Println("Failed to open log file:", logfile, err)
		os.Exit(-1)
	}
	jww.SetLogOutput(file)
	if verbose {
		jww.SetLogThreshold(jww.LevelInfo)
		jww.SetStdoutThreshold(jww.LevelFatal)
	}
}

func printCommand(cmd *exec.Cmd) {
	jww.INFO.Printf("==> Executing: %s\n", strings.Join(cmd.Args, " "))
}

func printError(err error) {
	if err != nil {
		jww.ERROR.Println(fmt.Sprintf("==> Error: %s\n", err.Error()))
	}
}
