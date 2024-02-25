package network

import (
	"Mydockker/meta"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

/**
 * 桥接网络驱动
 */

type BridgeNetworkDriver struct {
}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

/**
 * 创建桥接网络
 */
func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	ip, netNum, _ := net.ParseCIDR(subnet)
	netNum.IP = ip
	n := &Network{
		Name:    name,
		IPRange: netNum,
		Driver:  d.Name(),
	}
	// 创建 Bridge 虚拟网络设备
	if err := d.initBridge(n); err != nil {
		return nil, meta.NewError(meta.NewErrorCode(meta.ErrDriverExec, meta.NETWORK), fmt.Sprintf("create bridgeNetwork %s failed", name), err)
	}
	return n, nil
}

/**
 * 删除桥接网络
 */
func (d *BridgeNetworkDriver) Delete(network *Network) error {
	link, err := netlink.LinkByName(network.Name)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrLink, meta.NETWORK), fmt.Sprintf("retrieving link %s failed", network.Name), err)
	}
	return netlink.LinkDel(link)
}

/**
 * 桥接网络连接——连接网桥和虚拟网络设备端点
 */
func (d *BridgeNetworkDriver) Connect(network *Network, endPoint *EndPoint) error {
	bridgeName := network.Name
	brLink, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrLink, meta.NETWORK), fmt.Sprintf("retrieving link %s failed", network.Name), err)
	}
	// veth-bridge 端配置Connect
	veAttr := netlink.NewLinkAttrs()
	veAttr.Name = endPoint.ID[:5]
	veAttr.MasterIndex = brLink.Attrs().Index
	// 封装 EndPoint 设备
	endPoint.Device = netlink.Veth{
		LinkAttrs: veAttr,
		// veth-container 端设备名
		PeerName: "cif-" + endPoint.ID[:5],
	}
	// 添加 veth-bridge 设备一端到系统中
	if err := netlink.LinkAdd(&endPoint.Device); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrLink, meta.NETWORK), fmt.Sprintf("add Endpoint device %s failed", veAttr.Name), err)
	}
	// 启动 veth-bridge 设备
	if err := netlink.LinkSetUp(&endPoint.Device); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrLink, meta.NETWORK), fmt.Sprintf("setUp Endpoint device %s failed", veAttr.Name), err)
	}
	return nil
}

/**
 * 断开网络连接
 */
func (d *BridgeNetworkDriver) Disconnect(network Network, endpoint *EndPoint) error {
	return nil
}

/**
 * 初始化 Bridge 网络设备
 * 1.创建 Bridge 虚拟网络设备；
 * 2.配置 Bridge 虚拟网络设备地址和路由；
 * 3.启动 Bridge 设备；
 * 4.设置 iptables SNAT 规则；
 */
func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	bridgeName := n.Name
	// 1.创建 Bridge 虚拟设备
	if err := createBridgeInterface(bridgeName); err != nil {
		log.Error(err)
		return err
	}
	// 2.设置 Bridge IP地址和路由
	gatewayIP := *n.IPRange
	gatewayIP.IP = n.IPRange.IP
	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		log.Error(err)
		return err
	}
	// 3.启动 Bridge 虚拟网络设备
	if err := setInterfaceUP(bridgeName); err != nil {
		log.Error(err)
		return err
	}
	// 4.设置 iptables SNAT 规则
	if err := setIPTables(bridgeName, n.IPRange); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

/**
 * 删除网桥虚拟网络设备
 */
func (d *BridgeNetworkDriver) deleteBridge(n *Network) error {
	bridgeName := n.Name
	link, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrLink, meta.NETWORK), fmt.Sprintf("retrieving link %s failed", bridgeName), err)
	}
	if err := netlink.LinkDel(link); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrLink, meta.NETWORK), fmt.Sprintf("delete link %s failed", bridgeName), err)
	}
	return nil
}

/**
 * 创建虚拟网络设备
 * ip link add xxx 命令
 */
func createBridgeInterface(bridgeName string) error {
	_, err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}
	// 创建网桥
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	// 创建 Link 属性创建网桥对象
	br := &netlink.Bridge{LinkAttrs: la}
	if err := netlink.LinkAdd(br); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrLink, meta.NETWORK), fmt.Sprintf("create bridge %s error", bridgeName), err)
	}
	return nil
}

/**
 * 分配虚拟设备IP
 * ip addr add xxx 命令
 */
func setInterfaceIP(name string, rawIP string) error {
	retries := 2
	var err error
	var iface netlink.Link
	for i := 0; i < retries; i++ {
		// 通过 LinkByName 找到网络接口
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		log.Debugf("error retrieving new bridge newlink link [ %s ]... retrying", name)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrLink, meta.NETWORK), fmt.Sprintf("link virtual network device %s", name), err)
	}
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrConvert, meta.NETWORK), fmt.Sprintf("parse ipNet %s failed", rawIP), err)
	}
	addr := &netlink.Addr{IPNet: ipNet}
	return netlink.AddrAdd(iface, addr)
}

/**
 * 启动虚拟设备
 */
func setInterfaceUP(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrLink, meta.NETWORK), fmt.Sprintf("retrieving link %s failed", name), err)
	}
	if err = netlink.LinkSetUp(link); err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrLink, meta.NETWORK), fmt.Sprintf("linkSetup %s failed", name), err)
	}
	return nil
}

/**
 * 配置 iptables SNAT 规则转换（容器访问外网）
 * iptables -t nat -A POSTROUTING -s 172.18.0.0/24 -o eth0 -j MASQUERADE
 * iptables -t nat -A POSTROUTING -s {subnet/mast} -o {deviceName} -j MASQUERADE
 */
func setIPTables(bridgeName string, subnet *net.IPNet) error {
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	_, err := cmd.Output()
	if err != nil {
		return meta.NewError(meta.NewErrorCode(meta.ErrIptables, meta.NETWORK), fmt.Sprintf("iptables %s snat change failed", bridgeName), err)
	}
	return err
}
