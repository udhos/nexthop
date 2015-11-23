package main

import (
	"fmt"
	"log"
	"strings"

	"command"
)

func installRibCommands(root *command.CmdNode) {
	command.CmdInstall(root, "list", command.EXEC, cmdList, "List command tree")
	command.CmdInstall(root, "quit", command.EXEC, cmdQuit, "Quit session")
	command.CmdInstall(root, "reload", command.ENAB, cmdReload, "Reload")
	command.CmdInstall(root, "reload", command.ENAB, cmdReload, "Ugh") // duplicated command
	command.CmdInstall(root, "show interface", command.EXEC, cmdShowInt, "Show interfaces")
	command.CmdInstall(root, "show", command.EXEC, cmdShowInt, "Ugh") // duplicated command
	command.CmdInstall(root, "show ip address", command.EXEC, cmdShowIPAddr, "Show addresses")
	command.CmdInstall(root, "show ip interface", command.EXEC, cmdShowIPInt, "Show interfaces")
	command.CmdInstall(root, "show ip interface detail", command.EXEC, cmdShowIPInt, "Show interface detail")
	command.CmdInstall(root, "show ip route", command.EXEC, cmdShowIPRoute, "Show routing table")
}

func cmdQuit(root *command.CmdNode, line string, c command.CmdClient) {
}

func list(node *command.CmdNode, depth int, c command.CmdClient) {
	handler := "----"
	if node.Handler != nil {
		handler = "LEAF"
	}
	ident := strings.Repeat(" ", 4*depth)
	//c.userOut <- fmt.Sprintf("%s %d %s[%s] desc=[%s]\r\n", handler, node.MinLevel, ident, node.Path, node.Desc)
	//sendln(c, fmt.Sprintf("%s %d %s[%s] desc=[%s]", handler, node.MinLevel, ident, node.Path, node.Desc))
	output := fmt.Sprintf("%s %d %s[%s] desc=[%s]\r\n", handler, node.MinLevel, ident, node.Path, node.Desc)
	log.Printf(output)
	for _, n := range node.Children {
		list(n, depth+1, c)
	}
}

func cmdList(root *command.CmdNode, line string, c command.CmdClient) {
	list(root, 0, c)
}

func cmdReload(root *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowInt(root *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowIPAddr(root *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowIPInt(root *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowIPRoute(root *command.CmdNode, line string, c command.CmdClient) {
}
