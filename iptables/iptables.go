package iptables

import (
	"fmt"
	utildbus "k8s.io/kubernetes/pkg/util/dbus"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	"k8s.io/utils/exec"
	"strings"
)

var (
	iptInterface utiliptables.Interface
)

const (
	chainName = "PUCKGO"

	filterInputRule = "-j " + chainName
	filterRule      = "-m addrtype --dst-type LOCAL -p tcp --dport %d -j REJECT"

	manglePreroutingRule = "-j " + chainName
	mangleRule           = "-i %s -p tcp -m tcp -j TPROXY --on-port %d --tproxy-mark 0x1/0x1"
)

func init() {
	execer := exec.New()

	dbus := utildbus.New()
	iptInterface = utiliptables.New(execer, dbus, utiliptables.ProtocolIpv4)
}

//iptables -I INPUT -m addrtype --dst-type LOCAL -p tcp --dport 1080 -j REJECT
func ensureFilterTable(dport int) error {
	iptInterface.EnsureRule(utiliptables.Prepend, utiliptables.TableFilter, utiliptables.ChainInput, strings.Split(filterInputRule, " ")...)
	rule := fmt.Sprintf(filterRule, dport)
	iptInterface.EnsureChain(utiliptables.TableFilter, chainName)
	iptInterface.FlushChain(utiliptables.TableFilter, chainName)
	_, err := iptInterface.EnsureRule(utiliptables.Prepend, utiliptables.TableFilter, chainName, strings.Split(rule, " ")...)
	return err
}

//-A PREROUTING -i switch0 -p tcp -m tcp -j TPROXY --on-port 1080 --on-ip 0.0.0.0 --tproxy-mark 0x1/0x1
func ensureMangleTable(port string, dport int) error {
	iptInterface.EnsureRule(utiliptables.Prepend, utiliptables.TableMangle, utiliptables.ChainPrerouting, strings.Split(manglePreroutingRule, " ")...)
	rule := fmt.Sprintf(mangleRule, port, dport)
	iptInterface.EnsureChain(utiliptables.TableMangle, chainName)
	iptInterface.FlushChain(utiliptables.TableMangle, chainName)
	_, err := iptInterface.EnsureRule(utiliptables.Prepend, utiliptables.TableMangle, chainName, strings.Split(rule, " ")...)
	return err

}

// EnsureIptables create iptables rules for transparent mode proxy
func EnsureIptables(port string, listenPort int) error {
	if err := ensureFilterTable(listenPort); err != nil {
		return err
	}
	return ensureMangleTable(port, listenPort)
}
