package command

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

func OnlyExec(cmdStr string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", cmdStr)
	println(cmd.String())
	buf, err := cmd.CombinedOutput()
	println(string(buf))
	return string(buf), err
}

func ExecResultStr(cmdStr string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", cmdStr)
	println(cmd.String())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	defer stdout.Close()
	if err := cmd.Start(); err != nil {
		return "", err
	}

	buf, err := io.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	return string(buf), cmd.Wait()
}

// exec smart
func ExecSmartCTLByPath(path string) []byte {
	timeout := 6
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	//smartctl -i -n standby /dev/sdc  TODO:https://www.ippa.top/956.html
	cmd := exec.CommandContext(ctx, "smartctl", "-a", "-n", "standby", path, "-j")
	println(cmd.String())

	output, err := cmd.Output()
	if err != nil {
		fmt.Println(string(output))
		return nil
	}
	return output
}

func ExecEnabledSMART(path string) ([]byte, error) {
	return exec.Command("smartctl", "-s", "on", path).CombinedOutput()
}

// 执行 lsblk 命令
func ExecLSBLKByPath(path string) []byte {
	output, err := exec.Command("lsblk", path, "-O", "-J", "-b").Output()
	if err != nil {
		fmt.Println("lsblk", err)
		return nil
	}
	return output
}

// 执行 lsblk 命令
func ExecLSBLK() []byte {
	output, err := exec.Command("lsblk", "-O", "-J", "-b").Output()
	if err != nil {
		fmt.Println("lsblk", err)
		return nil
	}
	return output
}
