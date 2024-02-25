package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	log "github.com/sirupsen/logrus"
)

// 容器信息记录
type Info struct {
	Pid         string   `json:"pid"`         //容器进程Id
	Id          string   `json:"id"`          //容器Id
	Name        string   `json:"name"`        //容器名
	Command     string   `json:"command"`     //容器内init进程运行的命令
	CreateTime  string   `json:"createTime"`  //容器创建时间
	Status      string   `json:"status"`      //容器状态
	Volume      string   `json:"volume"`      //容器挂载的数据卷
	PortMapping []string `json:"portmapping"` //容器内端口映射
}

/**
 * start a new process, return executable commands
 * 1.use /proc/self/exe to create child process which diving by namespace and other environment;
 * 2.use init command param to init child process;
 * 3.redirect input/output/errput;
 *
 * perf:
 * 1.use pipe to transfer parameters between parentProcess and childProcess. Avoid out-of-buffer and console parameters too long
 */
func NewParentProcess(tty bool, volume, containerName, imageName string, envSlice []string) (*exec.Cmd, *os.File) {
	// create Pipe which transferring parameters between parentProcess and childProcess
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		log.Errorf("container_process::NewParentProcess new pipe failed")
		return nil, nil
	}
	// locate /proc/self/exe executable process
	exePath, err := os.Readlink("/proc/self/exe")
	if err != nil {
		log.Errorf("container_process::NewParentProcess can't find /proc/self/exe link")
		return nil, nil
	}
	processCmd := exec.Command(exePath, "init")
	// new process is divided by namespace
	processCmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWNET | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC,
	}
	// redirect output/input
	if tty {
		processCmd.Stdin = os.Stdin
		processCmd.Stdout = os.Stdout
		processCmd.Stderr = os.Stderr
	} else {
		// if allow process exec backgroundly, redirect output/input fd
		dirURL := fmt.Sprintf(InfoLogFormat, containerName)
		if err := os.MkdirAll(dirURL, Perm0622); err != nil {
			log.Errorf("container_process::NewParentProcess mkdir log directory failed %s", dirURL)
			return nil, nil
		}
		logPath := dirURL + LogFileName
		file, err := os.Create(logPath)
		if err != nil {
			log.Errorf("container_process::NewParentProcess create logFile failed %s", logPath)
			return nil, nil
		}
		processCmd.Stdout = file
	}
	// set readPipe、workingRootfs、environment for parentProcess
	processCmd.ExtraFiles = []*os.File{readPipe}
	processCmd.Dir = fmt.Sprintf(MergedDirFormat, containerName)
	processCmd.Env = append(os.Environ(), envSlice...)
	// create overlay2 fileSystem as container root workingspace
	NewWorkSpace(volume, imageName, containerName)
	return processCmd, writePipe
}
