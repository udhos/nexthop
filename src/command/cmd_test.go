package command

import (
	"testing"
)

func TestCmdInstall(t *testing.T) {

	cmdNone := CMD_NONE
	cmdConf := CMD_CONF

	root := &CmdNode{Path: "", MinLevel: EXEC, Handler: nil}

	cmdBogus := func(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	}

	if _, err := cmdAdd(root, cmdNone, "configure", ENAB, cmdBogus, "Enter configuration mode"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "enable", EXEC, cmdBogus, "Enter privileged mode"); err != nil {
		t.Errorf("error: %v", err)
	}
	descUnreach := "interface {IFNAME} description {ANY} unreachable"
	if _, err := cmdAdd(root, cmdConf, descUnreach, CONF, cmdBogus, "Interface description"); err == nil {
		t.Errorf("error: silently installed unreachable command location: [%s]", descUnreach)
	}
	if _, err := cmdAdd(root, cmdConf, "interface {IFNAME} ipv4 address {IPADDR}", CONF, cmdBogus, "Assign IPv4 address to interface"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdConf, "interface {IFNAME} ipv6 address {IPADDR6}", CONF, cmdBogus, "Assign IPv6 address to interface"); err != nil {
		t.Errorf("error: %v", err)
	}
	c := "interface {IFNAME} ip address {IPADDR}"
	if _, err := cmdAdd(root, cmdConf, c, CONF, cmdBogus, "Assign address to interface"); err == nil {
		t.Errorf("error: silently installed ambiguous command location: [%s]", c)
	}
	if _, err := cmdAdd(root, cmdConf, "ip routing", CONF, cmdBogus, "Enable IP routing"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdConf, "hostname HOSTNAME", CONF, cmdBogus, "Assign hostname"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "list", EXEC, cmdBogus, "List command tree"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "quit", EXEC, cmdBogus, "Quit session"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "reload", ENAB, cmdBogus, "Reload"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "reload", ENAB, cmdBogus, "Ugh"); err == nil {
		t.Errorf("error: silently reinstalled 'reload' command")
	}
	if _, err := cmdAdd(root, cmdNone, "rollback", CONF, cmdBogus, "Reset candidate configuration from active configuration"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "rollback {ID}", CONF, cmdBogus, "Reset candidate configuration from rollback configuration"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show interface", EXEC, cmdBogus, "Show interfaces"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show", EXEC, cmdBogus, "Ugh"); err == nil {
		t.Errorf("error: silently reinstalled 'show' command")
	}
	if _, err := cmdAdd(root, cmdNone, "show configuration", EXEC, cmdBogus, "Show candidate configuration"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show ip address", EXEC, cmdBogus, "Show addresses"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show ip interface", EXEC, cmdBogus, "Show interfaces"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show ip interface detail", EXEC, cmdBogus, "Show interface detail"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show ip route", EXEC, cmdBogus, "Show routing table"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show running-configuration", EXEC, cmdBogus, "Show active configuration"); err != nil {
		t.Errorf("error: %v", err)
	}
	if _, err := cmdAdd(root, cmdNone, "show version", EXEC, cmdBogus, "Show version"); err != nil {
		t.Errorf("error: %v", err)
	}
}
