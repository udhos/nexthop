package command

import (
	"testing"

	"fwd"
)

func TestCmdInstall(t *testing.T) {

	hardware := fwd.NewDataplaneBogus()

	listInterfaces := func() ([]string, []string) {
		ifaces, vrfs, err := hardware.Interfaces()
		if err != nil {
			t.Errorf("hardware.Interfaces(): error: %v", err)
		}
		return ifaces, vrfs
	}
	LoadKeywordTable(listInterfaces)

	cmdNone := CMD_NONE
	cmdConf := CMD_CONF

	root := &CmdNode{Path: "", MinLevel: EXEC, Handler: nil}

	cmdBogus := func(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	}

	if _, err := cmdAdd(root, cmdNone, "configure", ENAB, cmdBogus, nil, "Enter configuration mode"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "enable", EXEC, cmdBogus, nil, "Enter privileged mode"); err != nil {
		t.Errorf("error: %v", err)
	}
	descUnreach := "interface {IFNAME} description {ANY} unreachable"
	if _, err := cmdAdd(root, cmdConf, descUnreach, CONF, cmdBogus, ApplyBogus, "Interface description"); err == nil {
		t.Errorf("error: silently installed unreachable command location: [%s]", descUnreach)
	}
	if _, err := cmdAdd(root, cmdConf, "interface {IFNAME} ipv4 address {IFADDR}", CONF, cmdBogus, ApplyBogus, "Assign IPv4 address to interface"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdConf, "interface {IFNAME} ipv6 address {IFADDR6}", CONF, cmdBogus, ApplyBogus, "Assign IPv6 address to interface"); err != nil {
		t.Errorf("error: %v", err)
	}
	c := "interface {IFNAME} ip address {IFADDR}"
	if _, err := cmdAdd(root, cmdConf, c, CONF, cmdBogus, ApplyBogus, "Assign address to interface"); err == nil {
		t.Errorf("error: silently installed ambiguous command location: [%s]", c)
	}
	if _, err := cmdAdd(root, cmdConf, "ip routing", CONF, cmdBogus, ApplyBogus, "Enable IP routing"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdConf, "hostname HOSTNAME", CONF, cmdBogus, ApplyBogus, "Assign hostname"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "list", EXEC, cmdBogus, nil, "List command tree"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "quit", EXEC, cmdBogus, nil, "Quit session"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "reload", ENAB, cmdBogus, nil, "Reload"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "reload", ENAB, cmdBogus, nil, "Ugh"); err == nil {
		t.Errorf("error: silently reinstalled 'reload' command")
	}
	if _, err := cmdAdd(root, cmdNone, "rollback", CONF, cmdBogus, nil, "Reset candidate configuration from active configuration"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "rollback {ID}", CONF, cmdBogus, nil, "Reset candidate configuration from rollback configuration"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show interface", EXEC, cmdBogus, nil, "Show interfaces"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show", EXEC, cmdBogus, nil, "Ugh"); err == nil {
		t.Errorf("error: silently reinstalled 'show' command")
	}
	if _, err := cmdAdd(root, cmdNone, "show configuration", EXEC, cmdBogus, nil, "Show candidate configuration"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show ip address", EXEC, cmdBogus, nil, "Show addresses"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show ip interface", EXEC, cmdBogus, nil, "Show interfaces"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show ip interface detail", EXEC, cmdBogus, nil, "Show interface detail"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show ip route", EXEC, cmdBogus, nil, "Show routing table"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show running-configuration", EXEC, cmdBogus, nil, "Show active configuration"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show version", EXEC, cmdBogus, nil, "Show version"); err != nil {
		t.Errorf("error: %v", err)
	}
}
