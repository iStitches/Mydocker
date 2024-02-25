package main

import (
	"Mydockker/cgroups"
	"Mydockker/cgroups/subsystems"
	"Mydockker/container"
	"Mydockker/meta"
	"Mydockker/network"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

/**
 * clone process which dividing by namespace, using /proc/self/exe to init processResource
 * attention:
 * 1.only after childProcess has been inilizated that we can write message to writePipe by parentProcess
 */
func Run(tty bool, cmdArray []string, resConf *subsystems.ResourceConfig, volume string, containerName, imageName string,
	envSlice []string, nw string, portMapping []string) {
	// create containerId if containerName is null
	containerID := randStringBytes(container.IDLength)
	if containerName == "" {
		containerName = containerID
	}
	// get writePipe and initCmd of parentProcess
	cmdProcess, writePipe := container.NewParentProcess(tty, volume, containerName, imageName, envSlice)
	if cmdProcess == nil {
		log.Errorf("run::Run create child process failed")
		return
	}
	// create childProcess to init container
	if err := cmdProcess.Start(); err != nil {
		log.Errorf("run::Run parent Start failed %v", err)
		return
	}
	// record containerInfo
	if err := recordContainerInfo(cmdProcess.Process.Pid, cmdArray, containerName, containerID, volume); err != nil {
		log.Errorf("record containerInfo failed %v", err)
		return
	}
	// set resourceControl for container
	cgroupManager := cgroups.NewCgroupManger(meta.CGROUP_PATH)
	defer cgroupManager.Destory()
	cgroupManager.Set(resConf)
	cgroupManager.Apply(cmdProcess.Process.Pid, resConf)

	// set network-config for container
	if nw != "" {
		// init system-network
		network.Init()
		containerInfo := &container.Info{
			Id:          containerID,
			Name:        containerName,
			Pid:         strconv.Itoa(cmdProcess.Process.Pid),
			PortMapping: portMapping,
		}
		if err := network.Connect(nw, containerInfo); err != nil {
			log.Errorf("connect container %s and network %s failed: %v", containerName, nw, err)
			return
		}
	}

	// send parameters to childProcess after childProcess has been inilizated
	sendInitCommands(cmdArray, writePipe)
	if tty {
		_ = cmdProcess.Wait()
		container.DeleteWorkSpace(volume, containerName)
		deleteContainerInfo(containerName)
	}
}

/**
 * record containerInfo
 * 1）containerPid：容器进程ID；
 * 2）commandArray：容器命令行参数；
 * 3）containerName：容器名；
 * 4）containerId：容器ID；
 * 5）volume：容器挂载目录；
 */
func recordContainerInfo(containerPid int, commandArray []string, containerName, containerId, volume string) error {
	createTime := time.Now().Format("2006-01-02 15:04:05")
	command := strings.Join(commandArray, "")
	info := container.Info{
		Id:         containerId,
		Pid:        strconv.Itoa(containerPid),
		Command:    command,
		CreateTime: createTime,
		Name:       containerName,
		Status:     container.RUNNING,
		Volume:     volume,
	}
	jsonBytes, err := json.Marshal(info)
	if err != nil {
		log.Errorf("Record containerInfo is empty")
		return meta.NewError(meta.ErrWrite, "Record containerInfo is empty", err)
	}
	jsonStr := string(jsonBytes)
	// save containerInfo into local-file
	dirUrl := fmt.Sprintf(container.JsonFormat, containerName)
	if err := os.MkdirAll(dirUrl, container.Perm0622); err != nil {
		log.Errorf("Mkdir %s failed %v", dirUrl, err)
		return meta.NewError(meta.ErrWrite, "create containerInfo directory failed", err)
	}
	fileName := dirUrl + container.ConfigName
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.Errorf("Create file %s failed %v", fileName, err)
		return meta.NewError(meta.ErrWrite, "create containerInfo-file failed", err)
	}
	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("Write containerInfo into file failed %v", err)
		return meta.NewError(meta.ErrWrite, "write container-info into file failed", err)
	}
	return err
}

/**
 * delete containerInfo
 */
func deleteContainerInfo(containerName string) {
	url := fmt.Sprintf(container.JsonFormat, containerName)
	if err := os.RemoveAll(url); err != nil {
		log.Errorf("Remove containerInfo %v failed %v", url, err)
	}
}

/**
 *  send parameters to childProcess by pipe
 */
func sendInitCommands(commandArray []string, writePipe *os.File) {
	command := strings.Join(commandArray, " ")
	log.Infof("run::sendInitCommands all commands:%v", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

/**
 * create randStringBytes
 */
func randStringBytes(length int) string {
	letterTab := "01234567890abcdefghijklmnopqrstuvwxyz"
	rand.Seed(time.Now().UnixNano())
	ans := make([]byte, length)
	for i := 0; i < length; i++ {
		ans[i] = letterTab[rand.Intn(len(letterTab))]
	}
	return string(ans)
}
