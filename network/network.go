package network

import (
	"github.com/vishvananda/netlink"
	"net"
	"syscall"
)

const (
	//RTCF_LOCAL local route
	tableID       = 100
	destination   = "0.0.0.0/0"
	mark          = 0x1
	loopbackIndex = 1
)

func configRule(mark int, table int) error {
	rule := netlink.NewRule()
	rule.Mark = mark
	rule.Mark = mark
	rule.Table = table
	netlink.RuleDel(rule)
	return netlink.RuleAdd(rule)
}

func configDefaultRoute(table int) error {
	_, dst, err := net.ParseCIDR(destination)
	if err != nil {
		return err
	}
	route := &netlink.Route{Dst: dst, Scope: netlink.SCOPE_HOST, Type: syscall.RTN_LOCAL, Table: table, LinkIndex: loopbackIndex}
	netlink.RouteDel(route)
	return netlink.RouteAdd(route)
}

// ConfigTransparentNetwork config ip rule and ip route for transparent proxy
func ConfigTransparentNetwork() error {
	if err := configRule(mark, tableID); err != nil {
		return err
	}
	return configDefaultRoute(tableID)
}
