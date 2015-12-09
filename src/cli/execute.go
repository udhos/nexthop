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
	executeKey(ctx, line, c)
}

func executeKey(ctx command.ConfContext, line string, c *Client) {
	log.Printf("executeKey(): [%v]", line)

	if line == "q" {
		c.outputQueue = nil // discard output queue
	}

	c.Output() <- "\r\n"

	paging := c.SendQueue()
	c.SetSendEveryChar(paging)
	c.SendPrompt(ctx.Hostname(), paging)
	c.Flush()
}

func executeLine(ctx command.ConfContext, line string, c *Client) {
	log.Printf("executeLine: [%v]", line)

	status := c.Status()

	switch status {
	case command.MOTD:
		c.Sendln("")
		c.Sendln("rib server ready")
		c.Sendln("")
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
	c.SetSendEveryChar(paging)
	c.SendPrompt(ctx.Hostname(), paging)
	c.Flush()
}

func dispatchCommand(ctx command.ConfContext, line string, c *Client) {

	if line == "" {
		return
	}

	prependConfigPath := true // assume it's a config cmd

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
		c.Sendln(fmt.Sprintf("dispatchCommand: command not found: %s", err))
		return
	}

	if node.Handler == nil {
		c.Sendln(fmt.Sprintf("dispatchCommand: command missing handler: [%s]", lookupPath))
		if node.Options&command.CMD_CONF != 0 {
			c.ConfigPathSet(lookupPath)
		}
		return
	}

	if node.MinLevel > status {
		c.Sendln(fmt.Sprintf("dispatchCommand: command level prohibited: [%s]", lookupPath))
		return
	}

	node.Handler(ctx, node, lookupPath, c)
}
