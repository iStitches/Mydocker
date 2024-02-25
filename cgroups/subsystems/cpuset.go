package subsystems

import (
	"Mydockker/container"
	"Mydockker/meta"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

/**
 *  进程 CPU 资源配置:
 */
const CPU_APPLY_CONTROL_FILENAME = "cpuset.cpus"

type CpusetSubsystem struct {
}

func (c *CpusetSubsystem) Name() string {
	return "cpuset"
}

func (c *CpusetSubsystem) Set(cgroupPath string, conf *ResourceConfig) error {
	if conf.CpuSet == "" {
		return nil
	}
	subsysCgroupPath, err := getCgroupPath(c.Name(), cgroupPath, container.AutoCreate)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.CGROUPS), fmt.Sprintf("find base path of subsystem %s failed", c.Name()), err)
	}
	if err := ioutil.WriteFile(path.Join(subsysCgroupPath, CPU_APPLY_CONTROL_FILENAME), []byte(conf.CpuSet), container.Perm0644); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrWrite, meta.CGROUPS), fmt.Sprintf("set cgroup cpuset failed %v", err), err)
	}
	return nil
}

func (c *CpusetSubsystem) Apply(cgroupPath string, pid int, conf *ResourceConfig) error {
	if conf.CpuSet == "" {
		return nil
	}
	subsysCgroupPath, err := getCgroupPath(c.Name(), cgroupPath, true)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.CGROUPS), fmt.Sprintf("find base path of subsystem %s failed", c.Name()), err)
	}
	if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), container.Perm0644); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrWrite, meta.CGROUPS), fmt.Sprintf("set cgroup cpuset failed %v", err), err)
	}
	return nil
}

// 移除某个 cgroup
func (c *CpusetSubsystem) Remove(cgroupPath string) error {
	subsysCgroupPath, err := getCgroupPath(c.Name(), cgroupPath, true)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.CGROUPS), fmt.Sprintf("find base path of subsystem %s failed", c.Name()), err)
	}
	if err := os.RemoveAll(subsysCgroupPath); err != nil {
		return err
	}
	return nil
}
