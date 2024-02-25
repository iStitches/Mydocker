package network

import (
	"Mydockker/container"
	"Mydockker/meta"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	ipamDefaultAllocatorPath = "/var/run/Mydocker/network/ipam/subnet.json"
	defaultBufferSize        = 2000
)

/**
 * IP 地址分配管理器
 */
type IPAM struct {
	SubnetAllocPath string             // IP地址记录文件
	Subnets         *map[string]string // 位图，记录子网下所有可分配IP地址的情况
}

var ipAllocator = &IPAM{
	SubnetAllocPath: ipamDefaultAllocatorPath,
}

/**
 * 加载IP记录文件信息，初始化网络配置
 */
func (ipam *IPAM) load() error {
	if _, err := os.Stat((ipam.SubnetAllocPath)); err != nil {
		if !os.IsNotExist(err) {
			return meta.NewError(meta.NewErrorCode(meta.ErrRead, meta.NETWORK), fmt.Sprintf("stat ipamDefaultAllocatorFile %s failed", ipam.SubnetAllocPath), err)
		}
		return nil
	}
	file, err := os.Open(ipam.SubnetAllocPath)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrRead, meta.NETWORK), fmt.Sprintf("open ipamDefaultAllocatorFile %s failed", ipam.SubnetAllocPath), err)
	}
	defer file.Close()
	subnetJson := make([]byte, defaultBufferSize)
	n, err := file.Read(subnetJson)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrRead, meta.NETWORK), fmt.Sprintf("read ipamDefaultAllocatorFile %s failed", ipam.SubnetAllocPath), err)
	}
	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		return meta.NewError(meta.ErrConvert, fmt.Sprintf("unmarshal IPAM failed"), err)
	}
	return nil
}

/**
 * 持久化网络IP信息、网关信息
 */
func (ipam *IPAM) dump() error {
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if !os.IsNotExist(err) {
			return meta.NewError(meta.NewErrorCode(meta.NewErrorCode(meta.ErrRead, meta.NETWORK), meta.NETWORK), fmt.Sprintf("stat ipamDefaultAllocatorFile %s failed", ipam.SubnetAllocPath), err)
		}
		if err = os.MkdirAll(ipamConfigFileDir, container.Perm0644); err != nil {
			return meta.NewError(meta.NewErrorCode(meta.ErrRead, meta.NETWORK), fmt.Sprintf("mkdir ipamDefaultAllocatorFile %s failed", ipam.SubnetAllocPath), err)
		}
	}
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, container.Perm0644)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrRead, meta.NETWORK), fmt.Sprintf("open ipamDefaultAllocatorFile %s failed", ipam.SubnetAllocPath), err)
	}
	defer subnetConfigFile.Close()
	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return meta.NewError(meta.ErrConvert, fmt.Sprintf("marshal IPAM failed"), err)
	}
	_, err = subnetConfigFile.Write(ipamConfigJson)
	if err != nil {
		return meta.NewError(meta.ErrWrite, fmt.Sprintf("write ipamDefaultAllocatorFile %s failed", ipam.SubnetAllocPath), err)
	}
	return nil
}

/**
 * 在指定的网段下分配一个IP地址并通过位图记录
 */
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	ipam.Subnets = &map[string]string{}
	err = ipam.load()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	// 根据网段信息计算最多可定义多少种IP地址
	_, subnet, _ = net.ParseCIDR(subnet.String())
	use, all := subnet.Mask.Size()
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		// 1 << uint8(all-use) 表示网段中可用的IP地址数，初始化位图全部为0
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(all-use))
	}
	// 遍历位图数组，分配第一个可用IP地址
	for c := range (*ipam.Subnets)[subnet.String()] {
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			ipBytes := []byte((*ipam.Subnets)[subnet.String()])
			ipBytes[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipBytes)
			ip = subnet.IP
			// 根据位数序号在IP地址的每一位上累加，以获取下一个IP地址
			for t := uint(4); t > 0; t-- {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			// IP低位从1开始分配，0被网关占用
			ip[3] += 1
			break
		}
	}
	err = ipam.dump()
	if err != nil {
		log.Error(err)
	}
	return
}

/**
 * 释放网段下指定的IP地址
 */
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}
	_, subnet, _ = net.ParseCIDR(subnet.String())
	// 加载已存在的网络配置，释放对应IP
	if err := ipam.load(); err != nil {
		return err
	}
	// 根据IP找到位图数组的索引位置
	idx := 0
	releaseIP := ipaddr.To4()
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t-- {
		idx += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}
	// 重置对应位置为0
	ipBytes := []byte((*ipam.Subnets)[subnet.String()])
	ipBytes[idx] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipBytes)
	// 持久化
	if err := ipam.dump(); err != nil {
		log.Error(err)
	}
	return nil
}
