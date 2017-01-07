package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// RenameDirectory rename directory from old path to new name
func RenameDirectory(oldPath, newName string) (string, error) {
	newPath, err := ioutil.TempDir(filepath.Dir(oldPath), newName)
	if err != nil {
		return "", err
	}

	// os.Rename call fails on windows (https://github.com/golang/go/issues/14527)
	// Replacing with copyFolder to the newPath and deleting the oldPath directory
	if runtime.GOOS == "windows" {
		err = CopyFolder(oldPath, newPath)
		if err != nil {
			jww.ERROR.Printf("Error copying folder from: %s to: %s with error: %v", oldPath, newPath, err)
			return "", err
		}
		os.RemoveAll(oldPath)
		return newPath, nil
	}

	err = os.Rename(oldPath, newPath)
	if err != nil {
		return "", err
	}
	return newPath, nil
}

// CopyFolder Copy the folder from source to destination
func CopyFolder(source string, dest string) (err error) {
	fi, err := os.Lstat(source)
	if err != nil {
		jww.ERROR.Printf("Error getting stats for %s. %v", source, err)
		return err
	}

	err = os.MkdirAll(dest, fi.Mode())
	if err != nil {
		jww.ERROR.Printf("Unable to create %s directory %v", dest, err)
	}

	directory, _ := os.Open(source)

	defer directory.Close()

	objects, err := directory.Readdir(-1)

	for _, obj := range objects {
		if obj.Mode()&os.ModeSymlink != 0 {
			continue
		}

		sourcefilepointer := source + "\\" + obj.Name()
		destinationfilepointer := dest + "\\" + obj.Name()

		if obj.IsDir() {
			err = CopyFolder(sourcefilepointer, destinationfilepointer)
			if err != nil {
				return err
			}
		} else {
			err = CopyFile(sourcefilepointer, destinationfilepointer)
			if err != nil {
				return err
			}
		}

	}
	return
}

// CopyFile Copy file from source to destination
func CopyFile(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourcefile.Close()

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
			return err
		}

	}
	return
}
