package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/RTHeLL/ssh-keys-manager/internal/buildinfo"
)

const companionName = "sshkmcore"

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			fmt.Fprintf(os.Stdout, "version=%s commit=%s date=%s\n", buildinfo.Version, buildinfo.Commit, buildinfo.Date)
			_ = os.Stdout.Sync()
			return
		}
	}

	core := findCompanion()
	if core == "" {
		fmt.Fprintf(os.Stderr, "sshkm: не найден исполняемый файл %q рядом с sshkm и в PATH.\n", companionName)
		fmt.Fprintf(os.Stderr, "Соберите проект: make build (или go install ./cmd/sshkm ./cmd/sshkmcore).\n")
		os.Exit(127)
	}

	if err := execInto(core); err != nil {
		fmt.Fprintf(os.Stderr, "sshkm: %v\n", err)
		os.Exit(1)
	}
}

func findCompanion() string {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), companionName)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			return candidate
		}
	}
	if p, err := exec.LookPath(companionName); err == nil {
		return p
	}
	return ""
}

func execInto(core string) error {
	args := make([]string, len(os.Args))
	copy(args, os.Args)
	args[0] = core

	env := os.Environ()

	if runtime.GOOS == "windows" {
		cmd := exec.Command(core, args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env
		return cmd.Run()
	}

	if err := syscall.Exec(core, args, env); err != nil {
		cmd := exec.Command(core, args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env
		return cmd.Run()
	}
	// syscall.Exec при успехе не возвращает.
	return nil
}
