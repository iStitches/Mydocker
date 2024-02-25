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
 *  进程 CPU 资源限额配置，主要修改以下配置文件：
 *  1.cpu.cfs_period_us;
 *  2.cpu.cfs_quota_us
 */
const (
	CPU_PERIOD_CONTROL_FILENAME = "cpu.cfs_period_us"
	CPU_QUOTA_CONTROL_FILENAME  = "cpu.cfs_quota_us"
	CPU_SHARES_CONTROL_FILENAME = "cpu.shares"
	CPU_DEFAULT_PERIOD          = 100000
	CPU_DEFAULT_PERCENT         = 100
)

type CpuSubsystem struct {
}

func (c *CpuSubsystem) Name() string {
	return "cpu"
}

func (c *CpuSubsystem) Set(cgroupPath string, conf *ResourceConfig) error {
	if conf.CpuCfsQuota == 0 && conf.CpuShare == "" {
		return nil
	}
	subsysCgroupPath, err := getCgroupPath(c.Name(), cgroupPath, container.AutoCreate)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.CGROUPS), fmt.Sprintf("find base path of subsystem %s failed", c.Name()), err)
	}
	// cpu.shares 控制 CPU 的使用比例
	if conf.CpuShare != "" {
		if err := ioutil.WriteFile(path.Join(subsysCgroupPath, CPU_SHARES_CONTROL_FILENAME), []byte(conf.CpuShare), container.Perm0644); err != nil {
			return meta.NewError(meta.NewErrorCode(meta.ErrWrite, meta.CGROUPS), fmt.Sprintf("set cgroup cpu.shares failed %s", "cpushares"), err)
		}
	}
	// cpu.cfs_period_us、cpu.cfs_quota_us 控制 CPU 的使用时间
	if conf.CpuCfsQuota != 0 {
		// 配置总的 CPU 总时间
		if err = ioutil.WriteFile(path.Join(subsysCgroupPath, CPU_PERIOD_CONTROL_FILENAME), []byte(strconv.Itoa(CPU_DEFAULT_PERIOD)), container.Perm0644); err != nil {
			return meta.NewError(meta.NewErrorCode(meta.ErrWrite, meta.CGROUPS), fmt.Sprintf("set cgroup cpu.cfs_period_us failed %s", "cpushares"), err)
		}
		// 配置进程可以使用的时间片长度
		if err = ioutil.WriteFile(path.Join(subsysCgroupPath, CPU_QUOTA_CONTROL_FILENAME), []byte(strconv.Itoa(CPU_DEFAULT_PERIOD/CPU_DEFAULT_PERCENT*conf.CpuCfsQuota)), container.Perm0644); err != nil {
			return meta.NewError(meta.NewErrorCode(meta.ErrWrite, meta.CGROUPS), fmt.Sprintf("set cgroup cpu.cfs_quota_us failed %s", "cpuquota"), err)
		}
	}
	return nil
}

func (c *CpuSubsystem) Apply(cgroupPath string, pid int, conf *ResourceConfig) error {
	if conf.CpuCfsQuota == 0 && conf.CpuShare == "" {
		return nil
	}
	subsysCgroupPath, err := getCgroupPath(c.Name(), cgroupPath, true)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.CGROUPS), fmt.Sprintf("find base path of subsystem %s failed", c.Name()), err)
	}
	if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), container.Perm0644); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrWrite, meta.CGROUPS), fmt.Sprintf("set cgroup proc failed %v", err), err)
	}
	return nil
}

// 移除某个 cgroup
func (c *CpuSubsystem) Remove(cgroupPath string) error {
	subsysCgroupPath, err := getCgroupPath(c.Name(), cgroupPath, true)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.CGROUPS), fmt.Sprintf("find base path of subsystem %s failed", c.Name()), err)
	}
	if err := os.RemoveAll(subsysCgroupPath); err != nil {
		return err
	}
	return nil
}
