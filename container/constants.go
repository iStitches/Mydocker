package container

import "fmt"

const (
	RUNNING       = "running"
	STOP          = "stopped"
	Exit          = "exited"
	InfoLocation  = "/home/root/goproject/Mydocker/log/"
	InfoLogFormat = InfoLocation + "%s/"
	JsonLocation  = "/home/root/goproject/Mydocker/json/"
	JsonFormat    = JsonLocation + "%s/"
	ConfigName    = "config.json"
	LogFileName   = "container.log"
	IDLength      = 10
)

// container directory
const (
	RootUrl         = "/root/"
	MergedDirFormat = "/root/%s/merged"
	WorkDirFormat   = "/root/%s/work"
	LowerDirFormat  = "/root/%s/lower"
	UpperDirFormat  = "/root/%s/upper"
	OverlayFSFormat = "lowerdir=%s,upperdir=%s,workdir=%s"
)

// cgroup configuration
const (
	AutoCreate = false
)

const (
	Perm0777 = 0777 // all have read/write/exec permits
	Perm0755 = 0755 // only user has read/write/exec permits, other users have read/exec permits
	Perm0644 = 0644 // user has read/write permits, other users have read permits;
	Perm0622 = 0622 // user has read/write permits, other users have write permits;
)

func getImage(imageName string) string {
	return RootUrl + imageName + ".tar"
}

func getUnTar(imageName string) string {
	return RootUrl + imageName + "/"
}

func getLower(containerName string) string {
	return fmt.Sprintf(LowerDirFormat, containerName)
}

func getUpper(containerName string) string {
	return fmt.Sprintf(UpperDirFormat, containerName)
}

func getWorker(containerName string) string {
	return fmt.Sprintf(WorkDirFormat, containerName)
}

func getMerged(containerName string) string {
	return fmt.Sprintf(MergedDirFormat, containerName)
}
