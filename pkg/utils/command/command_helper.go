package command

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	exec2 "github.com/IceWhaleTech/CasaOS-Common/utils/exec"
)

// exec smart
func ExecSmartCTLByPath(path string) []byte {
	timeout := 6
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	// smartctl -i -n standby /dev/sdc  TODO:https://www.ippa.top/956.html
	cmd := exec2.CommandContext(ctx, "smartctl", "-a", "-n", "standby", path, "-j")

	output, err := cmd.Output()
	if err != nil {
		fmt.Println("smartctl", err.Error())
		fmt.Println("smartctl", string(path))
		fmt.Println("smartctl", len(output))
	}
	return output
}

func ExecEnabledSMART(path string) ([]byte, error) {
	return exec2.Command("smartctl", "-s", "on", path).CombinedOutput()
}

// 执行 lsblk 命令
func ExecLSBLKByPath(path string) []byte {
	output, err := exec2.Command("lsblk", path, "-O", "-J", "-b").Output()
	if err != nil {
		fmt.Println("lsblk", err)
		return nil
	}
	return output
}

// 执行 lsblk 命令
func ExecLSBLK() []byte {
	output, err := exec2.Command("lsblk", "-O", "-J", "-b").Output()
	if err != nil {
		fmt.Println("lsblk", err)
		return nil
	}
	return output
}

func ExecuteCommand(name string, arg ...string) ([]byte, error) {
	cmd := exec2.Command(name, arg...)
	println(cmd.String())

	out, err := cmd.Output()
	println(string(out))
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			message := string(exitError.Stderr)
			println(message)
			return nil, errors.New(message)
		}
		return nil, err
	}

	return out, nil
}
