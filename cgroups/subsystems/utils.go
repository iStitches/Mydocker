package subsystems

import (
	"Mydockker/container"
	"bufio"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

const baseMountPath = "/proc/self/mountinfo"
const pathIndex = 4

/**
 *  combine and return target cgroup path
 *  1.find path of target subsystem;
 *  2.if path not exists, create a new directory;
 */
func getCgroupPath(subsystem string, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRoot := findCgroupMountPoint(subsystem)
	absPath := path.Join(cgroupRoot, cgroupPath)
	if !autoCreate {
		return absPath, nil
	}
	_, err := os.Stat(absPath)
	if err != nil && os.IsNotExist(err) {
		err = os.Mkdir(absPath, container.Perm0755)
		return absPath, err
	}
	return absPath, nil
}

/**
 * find mountpoint of target subsystem
 * 1.using /proc/self/mountinfo to get filePath;
 */
func findCgroupMountPoint(Subsystem string) string {
	f, err := os.Open(baseMountPath)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		str := sc.Text()
		// for example:  	32 25 0:28 / /sys/fs/cgroup/memory rw,nosuid,nodev,noexec,relatime shared:14 - cgroup cgroup rw,seclabel,memory
		fields := strings.Split(str, " ")
		subsystemNames := strings.Split(fields[len(fields)-1], ",")
		for _, opt := range subsystemNames {
			// get target Subsystem path
			if opt == Subsystem {
				return fields[pathIndex]
			}
		}
	}
	if err := sc.Err(); err != nil {
		log.Error("read failed:", err)
		return ""
	}
	return ""
}
