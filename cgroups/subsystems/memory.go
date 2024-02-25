package subsystems

import (
	"Mydockker/container"
	"Mydockker/meta"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	log "github.com/sirupsen/logrus"
)

/**
 * 进程 memory 资源配置
 * 1.向 cgroup/memory.limit_in_bytes 文件中写入指定内存资源限制值；
 * 2.添加某个进程到 cgroup 中，也就是往对应的 tasks 文件中写入 pid；
 * 3.删除 cgroup 目录；
 */
const MEMORY_CONTROL_FILENAME = "memory.limit_in_bytes"

type MemorySubsystem struct {
}

func (m *MemorySubsystem) Name() string {
	return "memory"
}

func (m *MemorySubsystem) Set(cgroupPath string, conf *ResourceConfig) error {
	if conf.MemoryLimit == "" {
		return nil
	}
	subsysCgroupPath, err := getCgroupPath(m.Name(), cgroupPath, container.AutoCreate)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.CGROUPS), fmt.Sprintf("find base path of subsystem %s failed", m.Name()), err)
	}
	if err := ioutil.WriteFile(path.Join(subsysCgroupPath, MEMORY_CONTROL_FILENAME), []byte(conf.MemoryLimit), container.Perm0644); err != nil {
		return meta.NewError(meta.ErrWrite, fmt.Sprintf("set cgroup memory failed %v", err), err)
	}
	log.Infof("set cgroup memory for %s values %v", m.Name(), conf.MemoryLimit)
	return nil
}

func (m *MemorySubsystem) Apply(cgroupPath string, pid int, conf *ResourceConfig) error {
	if conf.MemoryLimit == "" {
		return nil
	}
	subsysCgroupPath, err := getCgroupPath(m.Name(), cgroupPath, true)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.CGROUPS), fmt.Sprintf("find base path of subsystem %s failed", m.Name()), err)
	}
	if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), container.Perm0644); err != nil {
		return meta.NewError(meta.ErrWrite, fmt.Sprintf("set cgroup memory failed %v", err), err)
	}
	return nil
}

// 移除某个 cgroup
func (m *MemorySubsystem) Remove(cgroupPath string) error {
	subsysCgroupPath, err := getCgroupPath(m.Name(), cgroupPath, true)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.CGROUPS), fmt.Sprintf("find base path of subsystem %s failed", m.Name()), err)
	}
	if err := os.RemoveAll(subsysCgroupPath); err != nil {
		return err
	}
	return nil
}
