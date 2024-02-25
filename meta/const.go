package meta

import "strconv"

type Category uint8

const (
	CGROUPS   Category = 0x01
	CONTAINER Category = 0x02
	LOG       Category = 0x03
	META      Category = 0x04
	NETWORK   Category = 0x05
	NSENTER   Category = 0x06
)

const CGROUP_PATH = "mydocker-cgroup"

func (ce Category) String() string {
	switch ce {
	case CGROUPS:
		return "cgroups"
	case CONTAINER:
		return "container"
	case LOG:
		return "log"
	case META:
		return "meta"
	case NETWORK:
		return "network"
	case NSENTER:
		return "nsenter"
	default:
		return "CATEGORY " + strconv.Itoa(int(ce))
	}
}
