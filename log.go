package main

import (
	"Mydockker/container"
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

/**
 * read container's log
 */
func LogContainer(containerName string) {
	logFileLocation := fmt.Sprintf(container.InfoLogFormat, containerName) + container.LogFileName
	file, err := os.Open(logFileLocation)
	if err != nil {
		log.Errorf("Open log container file %s failed %v", logFileLocation, err)
		return
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf("Read log container file %s failed %v", logFileLocation, err)
		return
	}
	_, err = fmt.Fprint(os.Stdout, string(content))
	if err != nil {
		log.Errorf("Log container Fprint failed %v", err)
		return
	}
}
