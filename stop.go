package main

import (
	"Mydockker/container"
	"Mydockker/meta"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func StopContainer(containerName string) {
	// get pid of containerProcess
	info, err := getContainerInfoByName(containerName)
	if err != nil {
		log.Errorf("Get containerInfo %s failed %v", containerName, err)
		return
	}
	pid, err := strconv.Atoi(info.Pid)
	if err != nil {
		log.Errorf("Convert containerPid %s failed %v", containerName, err)
		return
	}
	// send SIGTERM to containerProcess
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		log.Errorf("Send SIGTERM to %s failed %v", containerName, err)
		return
	}
	// update and cleanup containerStatus
	info.Status = container.STOP
	info.Pid = " "
	content, err := json.Marshal(info)
	if err != nil {
		log.Errorf("Json marshal %s failed %v", containerName, err)
		return
	}
	dirUrl := fmt.Sprintf(container.JsonFormat, containerName)
	configPath := dirUrl + container.ConfigName
	if err := ioutil.WriteFile(configPath, content, container.Perm0622); err != nil {
		log.Errorf("Write file %s failed %v", configPath, err)
	}
}

/**
 * get containerInfo by containerName
 */
func getContainerInfoByName(containerName string) (*container.Info, error) {
	dirUrl := fmt.Sprintf(container.JsonFormat, containerName)
	configFilePath := dirUrl + container.ConfigName
	content, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, meta.NewError(meta.ErrRead, "Read configFile failed", err)
	}
	var info container.Info
	if err = json.Unmarshal(content, &info); err != nil {
		return nil, meta.NewError(meta.ErrRead, "Unmarshal containerInfo failed", err)
	}
	return &info, nil
}

/**
 * remove unused container
 */
func RemoveContainer(containerName string) {
	info, err := getContainerInfoByName(containerName)
	if err != nil {
		log.Errorf("Get container %s info failed %v", containerName, err)
		return
	}
	if info.Status != container.STOP {
		log.Errorf("Can't remove running container")
		return
	}
	dirUrl := fmt.Sprintf(container.JsonFormat, containerName)
	if err := os.RemoveAll(dirUrl); err != nil {
		log.Errorf("Remove containerInfoFile %s failed", containerName)
		return
	}
	container.DeleteWorkSpace(info.Volume, containerName)
}
