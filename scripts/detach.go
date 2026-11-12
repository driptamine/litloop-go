package main

import (
	"fmt"
	"os"
	"syscall"
)

// detach starts a local development service independently of the invoking terminal.
func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: detach <binary> <log-file>")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(binary, logPath string) error {
	input, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}
	defer input.Close()
	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer logFile.Close()
	directory, err := os.Getwd()
	if err != nil {
		return err
	}
	process, err := os.StartProcess(binary, []string{binary}, &os.ProcAttr{Dir: directory, Env: os.Environ(), Files: []*os.File{input, logFile, logFile}, Sys: &syscall.SysProcAttr{Setsid: true}})
	if err != nil {
		return err
	}
	fmt.Println(process.Pid)
	return process.Release()
}
