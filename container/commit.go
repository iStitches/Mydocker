package container

import (
	"Mydockker/meta"
	"fmt"
	"os"
	"os/exec"
)

/**
 * commit and tar container fileSystem to ${imageName}.tar
 */
func Commit(containerName, imageName string) error {
	mntUrl := getMerged(containerName)
	imageUrl := getImage(imageName)
	_, err := os.Stat(imageUrl)
	if err == nil {
		return fmt.Errorf("file %s already exists", imageUrl)
	}
	if _, err := exec.Command("tar", "-zcf", imageUrl, "-C", mntUrl, ".").CombinedOutput(); err != nil {
		return meta.NewError(meta.ErrInvalidParam, fmt.Sprintf("tar folder %s failed", imageUrl), err)
	}
	return nil
}
