package main

import (
	"Mydockker/container"
	"Mydockker/meta"
	_ "Mydockker/nsenter"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// 控制 nsenter 里面的 mydocker_pid、mydocker_cmd 这两个Key，控制 setns 函数调用
const (
	EnvExecPid = "mydocker_pid"
	EnvExecCmd = "mydocker_cmd"
)

/**
 * exec EnterContainer function
 */
func EnterContainer(containerName string, comArray []string) {
	// check by environment
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		log.Errorf("ExecContainer getContainerPidByName failed %v", err)
		return
	}
	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// concat command and exec commands
	cmdStr := strings.Join(comArray, " ")
	log.Infof("ExecContainer pid:%v, cmds:%v", pid, cmdStr)
	// set environment to avoid exec repeatly and set commands into environments to transfer params
	_ = os.Setenv(EnvExecPid, pid)
	_ = os.Setenv(EnvExecCmd, cmdStr)
	containerEnvs := getEnvsByPid(pid)
	cmd.Env = append(os.Environ(), containerEnvs...)
	if err := cmd.Run(); err != nil {
		log.Errorf("ExecContainter %s failed %v", containerName, err)
	}
}

/**
 * get containerPid By containterName
 */
func getContainerPidByName(containerName string) (string, error) {
	// get containerInfo by containerName
	dirUrl := fmt.Sprintf(container.JsonFormat, containerName)
	configFilePath := dirUrl + container.ConfigName
	content, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return "", meta.NewError(meta.ErrRead, "read containerInfo failed", err)
	}
	var info container.Info
	if err := json.Unmarshal(content, &info); err != nil {
		return "", meta.NewError(meta.ErrConvert, "convert to ContainerInfo failed", err)
	}
	return info.Pid, nil
}

/**
 * get environments by pid
 */
func getEnvsByPid(pid string) []string {
	path := fmt.Sprintf("/proc/%s/environ", pid)
	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Read envionFile %s failed %v", path, err)
		return nil
	}
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs
}
