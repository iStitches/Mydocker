package container

import (
	"Mydockker/meta"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

/**
 * after create containerProcess, its the first process to init process's resource
 * 1.mount current process proc config;
 * 2.read commands from readPipe;
 * 3.execve run command to replace init process as first process;
 */
func ContainerResourceInit() error {
	// read parameters from readPipe
	cmdArrays := readUserCommands()
	if len(cmdArrays) == 0 {
		return errors.New("init::ContainerResourceInit userCommands is nil")
	}
	// proc mount
	mountProc()
	// execute user commands
	path, err := exec.LookPath(cmdArrays[0])
	if err != nil {
		log.Errorf("init::ContainerResourceInit exec lookPath failed, err=%v", err)
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.CONTAINER), "exec lookPath not found", err)
	}
	log.Infof("init::ContainerResourceInit execuatble path=%v", path)
	if err = syscall.Exec(path, cmdArrays[0:], os.Environ()); err != nil {
		log.Errorf("init::ContainerResourceInit exec failed, err=%v", err)
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.CONTAINER), "exec user commands", err)
	}
	return nil
}

/**
 * 1）mount proc fileSystem；
 * 2）mount rootfs;
 * mountFlags:
 * 	 1.syscall.MS_NOEXEC：本文件系统中不允许运行其它程序；
 *	 2.syscall.MS_NOSUID：本系统运行程序时不允许 set-user-id、set-group-id；
 *   3.syscall.MS_NODEV：mount默认都会携带；
 * systemd 加入 linux后，mount namespace 更新为 shared by default，所以必须显式声明 mount namespace 独立于宿主机
 */
func mountProc() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Errorf("Get current location failed %v", err)
		return
	}
	log.Infof("Current location is %s", pwd)
	// change mount transferMode for private
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		log.Errorf("mount default namespace failed, err = %v", err)
		return
	}
	// remount rootfs
	if err = privotRoot(pwd); err != nil {
		log.Errorf("reMount failed %v", err)
	}
	// mount proc
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	if err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), ""); err != nil {
		log.Errorf("mount proc failed, err = %v", err)
		return
	}
}

const readPipe = 3

/**
 * read userCommands from readPipe(3)
 * 0——stdin
 * 1——stdout
 * 2——stderr
 * 3——readPipe
 */
func readUserCommands() []string {
	pipe := os.NewFile(readPipe, "pipe")
	defer pipe.Close()
	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		log.Errorf("init readPipe failed, err=%v", err)
		return nil
	}
	return strings.Split(string(msg), " ")
}

/**
 * change rootfs of container
 */
func privotRoot(root string) error {
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return meta.NewError(meta.ErrMount, "reMount rootfs failed", err)
	}
	// create directory "rootfs/.pivot_root" to save old_rootfs
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, Perm0777); err != nil {
		return meta.NewError(meta.ErrWrite, "create old_pivotRoot directory failed", err)
	}
	// reMount rootfs to newRoot and saving old_rootfs directory
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return meta.NewError(meta.ErrMount, fmt.Sprintf("remount rootfs %s failed", root), err)
	}
	// change working directory
	if err := syscall.Chdir("/"); err != nil {
		return meta.NewError(meta.ErrConvert, "change working directory failed", err)
	}
	// unmount old_rootfs
	pivotDir = filepath.Join("/", ".pivot_root")
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return meta.NewError(meta.ErrUnMount, fmt.Sprintf("unmount old_rootfs %s failed", pivotDir), err)
	}
	return os.Remove(pivotDir)
}
