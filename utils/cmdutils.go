package utils

import (
	"os"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
)

func runAndWriteCommand(outName, cmdName string, args ...string) error {
	cmd := exec.Command(cmdName, args...)
	outfile, err := os.Create(outName)
	if err != nil {
		panic(err)
	}
	defer outfile.Close()
	cmd.Stdout = outfile

	err = cmd.Start()
	if err != nil {
		log.Warn("Error running cmd: " + cmdName + " " + strings.Join(args, " ") + " > " + outName)
		return err
	}
	cmd.Wait()
	return nil
}

func runCommand(silent bool, cmd string, args ...string) (string, error) {
	out, err := exec.Command(cmd, args...).Output()
	if !silent {
		log.Debug("%s\n", out)
	}
	if err != nil {
		log.Warn("Command unsuccessful: " + cmd + " " + strings.Join(args, " "))
		return "", err
	}
	return string(out), nil
}
