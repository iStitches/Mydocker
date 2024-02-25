package main

import (
	"Mydockker/cgroups/subsystems"
	"Mydockker/container"
	"Mydockker/network"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/urfave/cli"
)

/**
 * start procedure:
 * 1. user exec Mydocker run by hand;
 * 2. urfave/cli parse user Commands;
 * 3. call runCommand method to build cmds Object;
 * 4. NewParentProcess method return cmds Object to runCommand method;
 * 5. according to cmds paramters, /proc/self/exe init will execute mydocker command, which inilizates container's environment
 * 6. all init procedures end;
 */

/**
 * Usage: ./Mydocker run xxx -it /bin/bash
 * container start command
 */
var runCommand = cli.Command{
	Name: "run",
	Usage: `Create a container with namespace and cgroups limit
			mydocker run -it [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "it", // 简单起见，这里把 -i 和 -t 参数合并成一个
			Usage: "enable tty",
		},
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		cli.StringFlag{
			Name:  "mem", // 为了避免和 stress 命令的 -m 参数冲突 这里使用 -mem,到时候可以看下解决冲突的方法
			Usage: "memory limit",
		},
		cli.StringFlag{
			Name:  "cpu",
			Usage: "cpu quota",
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit",
		},
		cli.StringFlag{
			Name:  "v",
			Usage: "volume",
		},
		// 提供run后面的-name指定容器名字参数
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
		cli.StringSliceFlag{
			Name:  "e",
			Usage: "set environment",
		},
		cli.StringFlag{
			Name:  "net",
			Usage: "container network",
		},
		cli.StringSliceFlag{
			Name:  "p",
			Usage: "port mapping",
		},
	},
	/**
	 * parse commandline, tty represents allow bash windows
	 */
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container command")
		}
		// collect all userCommands
		var cmdArray []string
		for _, arg := range context.Args() {
			cmdArray = append(cmdArray, arg)
		}
		tty := context.Bool("it")
		detach := context.Bool("d")
		if tty && detach {
			return fmt.Errorf("can't execute container by tty and detach synchronizly")
		}
		containerName := context.String("name")
		envSlice := context.StringSlice("e")
		volume := context.String("v")
		network := context.String("net")
		portMapping := context.StringSlice("p")
		imageName := cmdArray[0]
		cmdArray = cmdArray[1:]
		// init resourceConfig for container
		resConfig := &subsystems.ResourceConfig{
			MemoryLimit: context.String("mem"),
			CpuCfsQuota: context.Int("cpu"),
			CpuSet:      context.String("cpuset"),
		}
		log.Infof("resConf:%v", resConfig)
		// start container process
		Run(tty, cmdArray, resConfig, volume, containerName, imageName, envSlice, network, portMapping)
		return nil
	},
}

/**
 * container inilization command
 */
var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	/**
	 * init process resource after create container
	 */
	Action: func(context *cli.Context) error {
		log.Infof("exec init command")
		return container.ContainerResourceInit()
	},
}

/**
 * usage: ./Mydocker commit containerName
 */
var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit container to image",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing imageName")
		}
		containerName := context.Args().Get(0)
		imageName := context.Args().Get(1)
		container.Commit(containerName, imageName)
		return nil
	},
}

/**
 * usage: ./Mydocker ps
 */
var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(context *cli.Context) error {
		ListContainers()
		return nil
	},
}

/**
 * Usage: ./Mydocker logs containerName
 */
var logCommand = cli.Command{
	Name:  "logs",
	Usage: "print logs of a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("please input your containerName")
		}
		containerName := context.Args().Get(0)
		LogContainer(containerName)
		return nil
	},
}

/**
 * Usage: ./Mydocker exec containerName commands
 */
var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into a container",
	Action: func(context *cli.Context) error {
		// check whether environment exists
		if os.Getenv(EnvExecPid) != "" {
			log.Infof("pid callback pid %v", os.Getegid())
			return nil
		}
		// Usage：./Mydocker exec containerName commands
		if len(context.Args()) < 2 {
			return fmt.Errorf("missing container name or command")
		}
		containerName := context.Args().Get(0)
		commandArray := context.Args().Tail()
		EnterContainer(containerName, commandArray)
		return nil
	},
}

/**
 * Usage: ./Mydocker stop containerName
 */
var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "stop a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing containerName, can't stop")
		}
		containerName := context.Args().Get(0)
		StopContainer(containerName)
		return nil
	},
}

/**
 * Usage: ./Mydocker rm containerName
 */
var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "remove unused containers",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		RemoveContainer(containerName)
		return nil
	},
}

/**
 * Usage:
 * ./Mydocker network create --driver bridge --subnet 192.168.0.0/24 testnet
 * 	1.use IPAM to get available ip-address and gateway-address;
 * 	2.use network-driver to init configuration of network and endPoint;
 * 	3.apply veth-device and bridge-device;
 *
 * ./Mydocker network list
 */
var networkCommand = cli.Command{
	Name:  "network",
	Usage: "container network commands",
	Subcommands: []cli.Command{
		{
			Name:  "create",
			Usage: "create a container network",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "driver",
					Usage: "network driver",
				},
				cli.StringFlag{
					Name:  "subnet",
					Usage: "subnet cidr",
				},
			},
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("missing network name")
				}
				// load network-configuration by dumpFile
				err := network.Init()
				if err != nil {
					return fmt.Errorf("network init failed %v", err)
				}
				err = network.CreateNetwork(context.String("driver"), context.String("subnet"), context.Args()[0])
				if err != nil {
					return fmt.Errorf("create network failed %v", err)
				}
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "list container network",
			Action: func(context *cli.Context) error {
				err := network.Init()
				if err != nil {
					return fmt.Errorf("network init failed %v", err)
				}
				network.ListNetwork()
				return nil
			},
		},
		{
			Name:  "remove",
			Usage: "remove container network",
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("missing network name")
				}
				network.Init()
				if err := network.DeleteNetwork(context.Args()[0]); err != nil {
					return fmt.Errorf("remove network %s configuration-file failed", context.Args()[0])
				}
				return nil
			},
		},
	},
}
