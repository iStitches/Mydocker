package cgroups

import (
	"Mydockker/cgroups/subsystems"
	"Mydockker/meta"
	"fmt"
)

/**
 * CgroupManager 统一管理各 subsystem
 * 1.添加当前进程到路径 path 下各 subsystem；
 * 2.更新路径 path 下各 subsystem 配置；
 * 3.删除 subsystem 约束；
 */

type CgroupManager struct {
	Path     string
	Resource *subsystems.ResourceConfig
}

func NewCgroupManger(path string) *CgroupManager {
	return &CgroupManager{
		Path: path,
	}
}

/**
 * 添加进程到 cgroup 节点（进程组）
 */
func (c *CgroupManager) Apply(pid int, conf *subsystems.ResourceConfig) error {
	for _, subsysIns := range subsystems.SubsystemIns {
		if err := subsysIns.Apply(c.Path, pid, conf); err != nil {
			return meta.NewError(meta.NewErrorCode(meta.ErrWrite, meta.CGROUPS), fmt.Sprintf("CgroupManger::Apply subsystem %s failed", subsysIns.Name()), err)
		}
	}
	return nil
}

/**
 * 更新 Cgroups 资源配置
 */
func (c *CgroupManager) Set(conf *subsystems.ResourceConfig) error {
	for _, subsysIns := range subsystems.SubsystemIns {
		if err := subsysIns.Set(c.Path, conf); err != nil {
			return meta.NewError(meta.NewErrorCode(meta.ErrRead, meta.CGROUPS), fmt.Sprintf("CgroupManger::Set new subsystem.ResourceConfig %s failed", subsysIns.Name()), err)
		}
	}
	return nil
}

/**
 * 销毁所有 Cgroups 配置
 */
func (c *CgroupManager) Destory() error {
	for _, subsysIns := range subsystems.SubsystemIns {
		if err := subsysIns.Remove(c.Path); err != nil {
			return meta.NewError(meta.NewErrorCode(meta.ErrWrite, meta.CGROUPS), fmt.Sprintf("CgroupManger::Destory subsystem.ResourceConfig %s failed", subsysIns.Name()), err)
		}
		//log.Infof("delete directory %s for cgroupManger %s", c.Path, subsysIns.Name())
	}
	return nil
}
