package subsystems

/**
 * 传递资源限制配置结构体，包括内存限制、CPU使用限制、CPU核心数限制
 */
type ResourceConfig struct {
	MemoryLimit string
	CpuCfsQuota int
	CpuShare    string
	CpuSet      string
}

/**
 * subsystem 参数配置接口
 * for Example：mydocker run -it -m 100m -cpuset 1 -cpushare 512 /bin/sh
 *
 * hierarchy：cgroup树结构，并通过虚拟文件系统的方式暴露给用户；
 * cgroup：cgroup树中的节点，用于控制节点中进程的资源占用；
 * subsystem：作用于 hierarchy 中的 cgroup节点，控制节点中进程的资源占用；
 */
type Subsystem interface {
	// 子系统配置名称（cpu/memory/cpuset）
	Name() string
	// 添加 Subsystem 到 Cgroup 节点
	Set(cgroupPath string, conf *ResourceConfig) error
	// 添加对进程的 subsystem 限制
	Apply(cgroupPath string, pid int, conf *ResourceConfig) error
	// 移除指定路径的 Cgroup
	Remove(cgroupPath string) error
}

/**
 *  subsystem 约束集合
 */
var SubsystemIns = []Subsystem{
	&CpuSubsystem{},
	&CpusetSubsystem{},
	&MemorySubsystem{},
}
