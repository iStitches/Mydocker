package network

import (
	"Mydockker/container"
	"Mydockker/meta"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

var (
	defaultNetworkPath = "/var/run/Mydocker/network/network/"
	// 系统网络驱动
	drivers = map[string]Driver{}
	// 系统网络
	networks = map[string]*Network{}
)

/**
 * 网络信息
 */
type Network struct {
	Name    string     // 网络名
	IPRange *net.IPNet // 地址段
	Driver  string     // 网络驱动名
}

/**
 * 网络端点：设备id、设备类型、设备IP地址、设备Mac地址、网络信息、端口映射
 */
type EndPoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	Network     *Network
	PortMapping []string
}

/**
 * 网络驱动：不同驱动对网络的创建、连接和销毁策略不同
 */
type Driver interface {
	Name() string
	Create(subnet string, name string) (*Network, error)
	Delete(network *Network) error
	Connect(network *Network, endpoint *EndPoint) error
	Disconnect(network Network, endpoint *EndPoint) error
}

func (nw *Network) dump(filePath string) error {
	if _, err := os.Stat(filePath); err != nil {
		if !os.IsNotExist(err) {
			return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.NETWORK), fmt.Sprintf("not found network-dumpFile %s", filePath), err)
		}
		if err = os.MkdirAll(filePath, container.Perm0644); err != nil {
			return meta.NewError(meta.NewErrorCode(meta.ErrWrite, meta.NETWORK), fmt.Sprintf("mkdir network-dumpFile %s", filePath), err)
		}
	}
	nwPath := path.Join(filePath, nw.Name)
	file, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, container.Perm0644)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrRead, meta.NETWORK), fmt.Sprintf("open network-dumpFile %s failed", nwPath), err)
	}
	defer file.Close()
	nwJson, err := json.Marshal(nw)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrConvert, meta.NETWORK), "marshal network-object failed", err)
	}
	_, err = file.Write(nwJson)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrWrite, meta.NETWORK), fmt.Sprintf("write network-dumpFile %s failed", nwPath), err)
	}
	return nil
}

func (nw *Network) remove(filePath string) error {
	fullPath := path.Join(filePath, nw.Name)
	if _, err := os.Stat(fullPath); err != nil {
		if !os.IsNotExist(err) {
			return meta.NewError(meta.NewErrorCode(meta.ErrRead, meta.NETWORK), fmt.Sprintf("stat network-dumpFile %s failed", fullPath), err)
		}
		return nil
	}
	return os.Remove(fullPath)
}

/**
 * 初始加载系统网络配置
 */
func Init() error {
	// 加载桥接网络驱动
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver
	// 创建网络配置文件目录
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if !os.IsNotExist(err) {
			return meta.NewError(meta.ErrRead, fmt.Sprintf("stat filePath %s failed", defaultNetworkPath), err)
		}
		if err = os.MkdirAll(defaultNetworkPath, container.Perm0644); err != nil {
			return meta.NewError(meta.ErrConvert, fmt.Sprintf("mkdir filePath %s failed", defaultNetworkPath), err)
		}
	}
	// 检查网络配置目录下文件，解析生成相应 Network 对象
	err := filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		_, nwName := path.Split(nwPath)
		nw := &Network{
			Name: nwName,
		}
		if err = nw.load(nwPath); err != nil {
			log.Errorf("load network-file %s failed %v", nwPath, err)
		}
		networks[nwName] = nw
		return nil
	})
	if err != nil {
		return meta.NewError(meta.ErrRead, fmt.Sprintf("walk network-file %s failed", defaultNetworkPath), err)
	}
	return nil
}

/**
 * 加载网络配置文件
 */
func (nw *Network) load(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return meta.NewError(meta.ErrRead, fmt.Sprintf("open network-file %s failed", filePath), err)
	}
	defer file.Close()
	nwJson := make([]byte, defaultBufferSize)
	n, err := file.Read(nwJson)
	if err != nil {
		return meta.NewError(meta.ErrRead, fmt.Sprintf("read network-file %s failed", filePath), err)
	}
	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		return meta.NewError(meta.ErrConvert, "unmarshal network-file failed", err)
	}
	return nil
}

/**
 * 创建网络对象并持久化存储
 */
func CreateNetwork(driver, subnet, name string) error {
	_, cidr, _ := net.ParseCIDR(subnet)
	// IPAM 获取可用 IP 地址
	ip, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrIpamExec, meta.NETWORK), fmt.Sprintf("alloc available-ipAddress for %s failed", subnet), err)
	}
	cidr.IP = ip
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrDriverExec, meta.NETWORK), fmt.Sprintf("driver %s exec failed", driver), err)
	}
	return nw.dump(defaultNetworkPath)
}

/**
 * 展示网络配置列表
 */
func ListNetwork() {
	writer := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(writer, "NAME\tIpRange\tDriver\n")
	for _, nw := range networks {
		fmt.Fprintf(writer, "%s\t%s\t%s\n",
			nw.Name,
			nw.IPRange.String(),
			nw.Driver,
		)
	}
	if err := writer.Flush(); err != nil {
		log.Errorf("Flush failed %v", err)
		return
	}
}

/**
 * 删除指定网络配置
 * 1.释放IPAM分配的ip地址；
 * 2.删除网络设备和驱动；
 * 3.删除网络相关配置文件；
 */
func DeleteNetwork(networkName string) error {
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("failed to find network %s", nw.Name)
	}
	// IPAM 释放 IP 地址
	if err := ipAllocator.Release(nw.IPRange, &nw.IPRange.IP); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrIpamExec, meta.NETWORK), fmt.Sprintf("remove network gateway ip %s failed", nw.IPRange.IP.String()), err)
	}
	// 删除网络设备驱动
	if err := drivers[nw.Driver].Delete(nw); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrDriverExec, meta.NETWORK), fmt.Sprintf("remove network driver %s failed", nw.Driver), err)
	}
	// 删除网络配置文件
	return nw.remove(defaultNetworkPath)
}

/**
 * Usage：./Mydocker run -net testnet -p 8080:80 xxxx
 * 连接容器到历史创建的网络
 */
func Connect(networkName string, info *container.Info) error {
	network, ok := networks[networkName]
	if !ok {
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.NETWORK), fmt.Sprintf("can't find network %s", networkName), nil)
	}
	// 分配容器内IP地址
	containerIp, err := ipAllocator.Allocate(network.IPRange)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrIpamExec, meta.NETWORK), fmt.Sprintf("allocate container networkIp %s failed", networkName), err)
	}
	// 创建网络端点
	point := &EndPoint{
		ID:          fmt.Sprintf("%s-%s", info.Id, networkName),
		IPAddress:   containerIp,
		Network:     network,
		PortMapping: info.PortMapping,
	}
	// 挂载 veth-bridge 设备
	if err = drivers[network.Driver].Connect(network, point); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrDriverExec, meta.NETWORK), fmt.Sprintf("veth-pair connect bridge and container failed, networkName: %s", networkName), err)
	}
	// 配置 veth-container 设备IP地址
	if err = configEndpointIpAddressAndRoute(point, info); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrLink, meta.NETWORK), "config veth-pair ip address of namespace failed", err)
	}
	// 配置容器端口和宿主机端口映射
	return configPortMapping(point)
}

/**
 * 配置容器网络端点（veth-container）的地址和路由
 */
func configEndpointIpAddressAndRoute(point *EndPoint, info *container.Info) error {
	// 找到容器端对应的veth设备
	containerLink, err := netlink.LinkByName(point.Device.PeerName)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrNotFound, meta.NETWORK), fmt.Sprintf("can't find veth device %s", point.Device.Name), err)
	}
	// 先进入容器的 net-namespace 环境中，在函数结束时恢复到宿主机的 net-namespace
	// 下面的操作都需要在容器的 net-namespace 环境下运行
	defer enterContainerNetNS(&containerLink, info)()
	// 获取容器的IP地址和网段，配置容器内部的接口地址
	ifaceIp := point.Network.IPRange
	ifaceIp.IP = point.IPAddress
	// 配置容器端veth设备的IP地址
	if err = setInterfaceIP(point.Device.PeerName, ifaceIp.String()); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrIpamExec, meta.NETWORK), fmt.Sprintf("set ip-address for veth-container %s failed", point.Device.PeerName), err)
	}
	// 启动容器内Veth端点
	if err = setInterfaceUP(point.Device.PeerName); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrDriverExec, meta.NETWORK), fmt.Sprintf("startUp veth-container %s failed", point.Device.PeerName), err)
	}
	// 配置本地回环地址
	if err = setInterfaceUP("lo"); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrDriverExec, meta.NETWORK), "startUp lo-device failed", err)
	}
	// 配置Namespace中容器网络的访问路由
	// route add -net 0.0.0.0/0 gw（Bridge网桥地址）dev（容器内Veth端点设备）
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	defaultRoute := &netlink.Route{
		LinkIndex: containerLink.Attrs().Index,
		Gw:        point.Network.IPRange.IP,
		Dst:       cidr,
	}
	// 添加路由到容器的网络空间
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrLink, meta.NETWORK), "add route to network failed", err)
	}
	return nil
}

/**
 * 添加容器的网络端点（veth-container设备）到容器Namespace，函数返回值为函数指针
 * 方法锁定当前程序所执行的线程，使当前线程进入容器的网络空间，只有执行返回的函数指针时才会退回到宿主机网络空间
 */
func enterContainerNetNS(endPointLink *netlink.Link, info *container.Info) func() {
	log.Infof("veth-container pid: %s", info.Pid)
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", info.Pid), os.O_RDONLY, 0)
	if err != nil {
		log.Errorf("get container net namespace failed %v", err)
	}
	nsFD := f.Fd()
	// 锁定当前 goroutine 到 kernelThread，防止 goroutine 调度到别的内核线程上去
	runtime.LockOSThread()
	// 加入 veth-container 设备到 container-net-Namespace 中
	if err = netlink.LinkSetNsFd(*endPointLink, int(nsFD)); err != nil {
		log.Errorf("set link netns failed %v", err)
	}
	// 获取原始 net-namespace
	origin, err := netns.Get()
	if err != nil {
		log.Errorf("get origin netNamespace failed: %v", err)
	}
	// 当前进程进入 container-net-namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		log.Errorf("set current process netNamespace failed:%v", err)
	}
	// 回退宿主机 net-namespace 函数操作
	return func() {
		netns.Set(origin)
		origin.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}

/**
 * 容器宿主机端口映射配置
 */
func configPortMapping(point *EndPoint) error {
	var err error
	for _, pm := range point.PortMapping {
		mappings := strings.Split(pm, ":")
		if len(mappings) != 2 {
			log.Errorf("container portMapping %s failed", pm)
			continue
		}
		// iptables 配置端口映射路由
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			mappings[0], point.IPAddress.String(), mappings[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		log.Infof("set portMapping for container, cmd: %s", cmd)
		output, err := cmd.Output()
		if err != nil {
			log.Errorf("set portMapping %s:%s failed, output:%s", mappings[0], mappings[1], output)
			continue
		}
	}
	return err
}
