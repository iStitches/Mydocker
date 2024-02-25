package container

import (
	"Mydockker/meta"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

/**
 * create an Overlay fileSystem as container root workingspace
 * 1）create lower-dir;
 * 2）create upper-dir、work-dir；
 * 3）create merged-dir and mount as overlayFS；
 * 4）mount volume if exists；
 */
func NewWorkSpace(volume, imageName, containerName string) {
	if err := createLower(imageName, containerName); err != nil {
		log.Error(err)
		return
	}
	if err := createDirs(containerName); err != nil {
		log.Error(err)
		return
	}
	if err := mountOverlayfs(containerName); err != nil {
		log.Error(err)
		return
	}
	if volume != "" {
		hostDir, containerDir, err := volumeUrlExtract(volume)
		if err != nil {
			log.Errorf("Invalid volume %s", volume)
			return
		}
		if err := mountVolume(containerName, []string{hostDir, containerDir}); err != nil {
			log.Errorf("Mount volume %s failed", volume)
			return
		}
	}
}

/**
 * parse mount information,
 */
func volumeUrlExtract(volume string) (sourcePath string, destinationPath string, err error) {
	parts := strings.Split(volume, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid volume [%s], split by %s", volume, ":")
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid volume [%s], path can't be empty", volume)
	}
	return parts[0], parts[1], nil
}

/**
 * mount volumes of parent-dir to container-dir
 */
func mountVolume(containerName string, volumes []string) error {
	// create parent-dir
	parentUrl := volumes[0]
	if err := os.Mkdir(parentUrl, Perm0755); err != nil {
		log.Infof("mkdir parent dir %s failed. %v", parentUrl, err)
	}
	containerUrl := volumes[1]
	// create container's exact mount-dir $mntPath/$containerUrl
	mntUrl := getMerged(containerName)
	containerVolumeUrl := mntUrl + "/" + containerUrl
	if err := os.Mkdir(containerVolumeUrl, Perm0755); err != nil {
		log.Errorf("mkdir container dir %s failed. %v", containerVolumeUrl, err)
		return fmt.Errorf("Mkdir container-dir %s failed", containerVolumeUrl)
	}
	// bind mount parent-dir to container-dir
	// Usage: mount -o bind /hostUrl /containerUrl
	cmd := exec.Command("mount", "-o", "bind", parentUrl, containerVolumeUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("mount volume failed. %v", err)
	}
	log.Infof("mountVolume from %s to %s successfully", parentUrl, containerVolumeUrl)
	return nil
}

/**
 * create readOnly directory of lower-dir
 */
func createLower(imageName, containerName string) error {
	// concat imagePath and target-untar position
	imageUrl := getImage(imageName)
	lower := getLower(containerName)
	_, err := os.Stat(lower)
	if err != nil && os.IsNotExist(err) {
		log.Warnf("lower-dir %s not exists, imageTarUrl %s", lower, imageUrl)
		if err = os.MkdirAll(lower, Perm0622); err != nil {
			return meta.NewError(meta.ErrWrite, fmt.Sprintf("Create lower-dir %s failed", lower), err)
		}
		if _, err := exec.Command("tar", "-xvf", imageUrl, "-C", lower).CombinedOutput(); err != nil {
			return meta.NewError(meta.ErrWrite, fmt.Sprintf("Untar imageTar %s failed", imageUrl), err)
		}
	}
	return nil
}

/**
 * create upper-dir and work-dir of overlayFS
 */
func createDirs(containerName string) error {
	upperUrl := getUpper(containerName)
	if err := os.MkdirAll(upperUrl, Perm0755); err != nil {
		return meta.NewError(meta.ErrWrite, fmt.Sprintf("Create upper-dir %s failed", upperUrl), err)
	}
	workUrl := getWorker(containerName)
	if err := os.Mkdir(workUrl, Perm0755); err != nil {
		return meta.NewError(meta.ErrWrite, fmt.Sprintf("Create work-dir %s failed", workUrl), err)
	}
	return nil
}

/**
 * mountOverlayFS
 * mount -t overlay overlay -o lowerdir=lower1:lower2:lower3,upperdir=upper,workdir=work merged
 */
func mountOverlayfs(containerName string) error {
	// create mount-url
	mntUrl := fmt.Sprintf(MergedDirFormat, containerName)
	if err := os.MkdirAll(mntUrl, Perm0777); err != nil {
		return meta.NewError(meta.ErrWrite, fmt.Sprintf("Mkdir mntUrl %s failed", mntUrl), err)
	}
	// combine arguments
	// e.g. lowerdir=/root/busybox,upperdir=/root/upper,workdir=/root/work
	var (
		lower  = getLower(containerName)
		upper  = getUpper(containerName)
		worker = getWorker(containerName)
		merged = getMerged(containerName)
	)
	dirs := getOverlayFSDirs(lower, upper, worker)
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, merged)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	log.Infof("mount overlayfs: [%s]", cmd.String())
	if err := cmd.Run(); err != nil {
		return meta.NewError(meta.ErrMount, "Mount overlayfs failed", err)
	}
	return nil
}

/**
 * get overlay fileSystem dir
 */
func getOverlayFSDirs(lower, upper, worker string) string {
	return fmt.Sprintf(OverlayFSFormat, lower, upper, worker)
}

/**
 * Delete overlayfs workingPlace
 * 1）uninstall volume；
 * 2）uninstall and delete merged directory；
 * 3）uninstall and delete upper-dir、work-dir；
 */
func DeleteWorkSpace(volume, containerName string) error {
	log.Infof("DeleteWorkSpace, volume:%s, containerName:%s", volume, containerName)
	if volume != "" {
		_, containerPath, err := volumeUrlExtract(volume)
		if err != nil {
			return meta.NewError(meta.ErrRead, fmt.Sprintf("Extract volume failed, volume : %s", volume), err)
		}
		mntPath := getMerged(containerName)
		if err := unmountVolume(mntPath, containerPath); err != nil {
			return meta.NewError(meta.ErrUnMount, fmt.Sprintf("UnmountVolume %s failed", mntPath+containerPath), err)
		}
	}
	if err := unmountOverlayfs(containerName); err != nil {
		log.Error(err)
		return err
	}
	if err := removeDirs(containerName); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

/**
 * umount volume of parentDir
 */
func unmountVolume(mntUrl, containerPath string) error {
	containerPathInHost := path.Join(mntUrl, containerPath)
	cmd := exec.Command("umount", containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return meta.NewError(meta.ErrUnMount, fmt.Sprintf("umountVolume %s failed", containerPathInHost), err)
	}
	return nil
}

/**
 * unmount merged-dir of overlayfs
 */
func unmountOverlayfs(containerName string) error {
	mntUrl := getMerged(containerName)
	cmd := exec.Command("umount", mntUrl)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return meta.NewError(meta.ErrUnMount, fmt.Sprintf("Umount Overlayfs %s failed", mntUrl), err)
	}
	if err := os.RemoveAll(mntUrl); err != nil {
		return fmt.Errorf("Remove mountDir %s failed", mntUrl)
	}
	log.Infof("volume::unmountOverlayfs unmountFS %s successfully", mntUrl)
	return nil
}

/**
 * remove directories(merged-dir、upper-dir、work-dir) but save (lower-dir)
 */
func removeDirs(containerName string) error {
	lower := getLower(containerName)
	upper := getUpper(containerName)
	worker := getWorker(containerName)
	if err := os.RemoveAll(upper); err != nil {
		return fmt.Errorf("Remove lower-dir %s failed", upper)
	}
	if err := os.RemoveAll(worker); err != nil {
		return fmt.Errorf("Remove worker-dir %s failed", worker)
	}
	if err := os.RemoveAll(lower); err != nil {
		return fmt.Errorf("Remove lower-dir %s failed", worker)
	}
	log.Infof("volume::removeDirs upper-dir %s work-dir %s lower-dir %s successfully", upper, worker, lower)
	return nil
}
