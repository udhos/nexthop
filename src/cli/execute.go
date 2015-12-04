package cli

import (
	"fmt"
	"log"

	"command"
)

func Execute(ctx command.ConfContext, line string, isLine bool, c *Client) {
	log.Printf("cli.Execute: isLine=%v cmd=[%s]", isLine, line)

	if isLine {
		// full-line command
		executeLine(ctx, line, c)
		return
	}

	// single-char command
	log.Printf("cli.Execute: isLine=%v cmd=[%s] single-char command", isLine, line)
}

func executeLine(ctx command.ConfContext, line string, c *Client) {
	log.Printf("executeLine: [%v]", line)

	status := c.Status()

	switch status {
	case command.MOTD:
		c.Sendln("")
		c.Sendln("rib server ready")
		c.StatusSet(command.USER)
	case command.USER:
		c.EchoDisable()
		c.StatusSet(command.PASS)
	case command.PASS:
		c.EchoEnable()
		c.StatusSet(command.EXEC)
	case command.EXEC, command.ENAB, command.CONF:
		dispatchCommand(ctx, line, c)
	default:
		msg := fmt.Sprintf("unknown state for command: [%s]", line)
		log.Print(msg)
		c.Sendln(msg)
	}

	paging := c.SendQueue()
	//c.SetSendEveryChar(!paging)
	c.SendPrompt(paging)
	c.Flush()

	log.Printf("executeLine: flushed [%v]", line)
}

func dispatchCommand(ctx command.ConfContext, line string, c *Client) {

	/*
		if line == "" {
			return
		}
	*/

	prependConfigPath := true

	status := c.Status()

	n, e := command.CmdFind(ctx.CmdRoot(), line, status)
	if e == nil {
		// found at root
		if n.Options&command.CMD_CONF == 0 {
			// not a config cmd
			prependConfigPath = false
		}
	}

	lookupPath := line
	configPath := c.ConfigPath()
	if prependConfigPath && configPath != "" {
		lookupPath = fmt.Sprintf("%s %s", c.ConfigPath(), line)
	}

	//log.Printf("dispatchCommand: prepend=%v path=[%s] line=[%s] full=[%s]", prependConfigPath, configPath, line, lookupPath)

	node, err := command.CmdFind(ctx.CmdRoot(), lookupPath, status)
	if err != nil {
		log.Printf("dispatchCommand: command not found: %s", err)
		return
	}

	if node.Handler == nil {
		log.Printf("dispatchCommand: command missing handler: [%s]", lookupPath)
		if node.Options&command.CMD_CONF != 0 {
			c.ConfigPathSet(lookupPath)
		}
		return
	}

	if node.MinLevel > status {
		log.Printf("dispatchCommand: command level prohibited: [%s]", lookupPath)
		return
	}

	node.Handler(ctx, node, lookupPath, c)
}
