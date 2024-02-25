package main

import (
	"Mydockker/container"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"text/tabwriter"

	log "github.com/sirupsen/logrus"
)

/**
 * read information of running containerProcess
 */
func ListContainers() {
	files, err := ioutil.ReadDir(container.JsonLocation)
	if err != nil {
		log.Errorf("read containerInfo %s failed %v", container.InfoLocation, err)
		return
	}
	containers := make([]*container.Info, 0, len(files))
	for _, file := range files {
		tmpInfo, err := getContainerInfo(file)
		if err != nil {
			log.Errorf("read containerInfo %v failed", file.Name())
			continue
		}
		containers = append(containers, tmpInfo)
	}
	// print containerInfos into console
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	_, err = fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	if err != nil {
		log.Errorf("Fprint error %v", err)
	}
	for _, item := range containers {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreateTime)
		if err != nil {
			log.Errorf("Fprint error %v", err)
		}
	}
	if err := w.Flush(); err != nil {
		log.Errorf("Flush failed %v", err)
	}
}

/**
 * read container information
 */
func getContainerInfo(file os.FileInfo) (*container.Info, error) {
	// concat containerFile with containerName
	containerName := file.Name()
	configFileDir := fmt.Sprintf(container.JsonFormat, containerName)
	configFileDir = configFileDir + container.ConfigName
	content, err := ioutil.ReadFile(configFileDir)
	if err != nil {
		log.Errorf("read containerFile %s failed %v", configFileDir, err)
		return nil, err
	}
	info := new(container.Info)
	if err := json.Unmarshal(content, info); err != nil {
		log.Errorf("json unmarshal failed %v", err)
		return nil, err
	}
	return info, nil
}
