package command

import (
	//"fmt"
	"testing"
)

func TestCmdInstall(t *testing.T) {

	root := &CmdNode{Path: "", MinLevel: EXEC, Handler: nil}

	cmdBogus := func(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	}

	if err := cmdAdd(root, "configure", ENAB, cmdBogus, "Enter configuration mode"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "enable", EXEC, cmdBogus, "Enter privileged mode"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "interface {IFNAME} ipv4 address {IPADDR}", CONF, cmdBogus, "Assign address to interface"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "interface {IFNAME} ipv6 address {IPADDR6}", CONF, cmdBogus, "Assign IPv6 address to interface"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "ip routing", CONF, cmdBogus, "Enable IP routing"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "hostname HOSTNAME", CONF, cmdBogus, "Assign hostname"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "list", EXEC, cmdBogus, "List command tree"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "quit", EXEC, cmdBogus, "Quit session"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "reload", ENAB, cmdBogus, "Reload"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "reload", ENAB, cmdBogus, "Ugh"); err == nil {
		t.Errorf("error: silently reinstalled 'reload' command")
	}
	if err := cmdAdd(root, "show interface", EXEC, cmdBogus, "Show interfaces"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "show", EXEC, cmdBogus, "Ugh"); err == nil {
		t.Errorf("error: silently reinstalled 'show' command")
	}
	if err := cmdAdd(root, "show configuration", EXEC, cmdBogus, "Show candidate configuration"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "show ip address", EXEC, cmdBogus, "Show addresses"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "show ip interface", EXEC, cmdBogus, "Show interfaces"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "show ip interface detail", EXEC, cmdBogus, "Show interface detail"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "show ip route", EXEC, cmdBogus, "Show routing table"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "show running-configuration", EXEC, cmdBogus, "Show active configuration"); err != nil {
		t.Errorf("error: %v", err)
	}
	if err := cmdAdd(root, "show version", EXEC, cmdBogus, "Show version"); err != nil {
		t.Errorf("error: %v", err)
	}
}
