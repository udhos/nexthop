package cli

import (
	"fmt"
	"log"
	"strings"

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

	// key feedback
	// RETURN is empty line (line == "")
	c.Output() <- fmt.Sprintf("%s\r\n", line)

	commandFeedback(c, ctx.Hostname())
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

	commandFeedback(c, ctx.Hostname())
}

func commandFeedback(c *Client, hostname string) {
	paging := c.SendQueue()
	c.SetSendEveryChar(paging)
	c.SendPrompt(hostname, paging)
	c.Flush()
}

func dispatchCommand(ctx command.ConfContext, rawLine string, c *Client) {

	line := strings.TrimLeft(rawLine, " ")

	if line == "" {
		return // ignore empty lines
	}

	prependConfigPath := true // assume it's a config cmd

	status := c.Status()

	n, e := command.CmdFind(ctx.CmdRoot(), line, status)
	if e == nil {
		// found at root
		if n.Options&command.CMD_CONF == 0 {
			// not a config cmd -- ignore prepend path
			prependConfigPath = false
		}
	}

	lookupPath := line
	configPath := c.ConfigPath()
	if prependConfigPath && configPath != "" {
		// prepend path to config command
		lookupPath = fmt.Sprintf("%s %s", c.ConfigPath(), line)
	}

	//log.Printf("dispatchCommand: prepend=%v path=[%s] line=[%s] full=[%s]", prependConfigPath, configPath, line, lookupPath)

	node, err := command.CmdFind(ctx.CmdRoot(), lookupPath, status)
	if err != nil {
		c.Sendln(fmt.Sprintf("dispatchCommand: command not found: %s", err))
		return
	}

	//c.Sendln(fmt.Sprintf("dispatchCommand: status=%d privilege=%d: [%s]", status, node.MinLevel, lookupPath))
	if node.MinLevel > status {
		c.Sendln(fmt.Sprintf("dispatchCommand: command level prohibited: [%s]", lookupPath))
		return
	}

	if node.Handler == nil {
		if node.Options&command.CMD_CONF != 0 {
			c.ConfigPathSet(lookupPath)
			return
		}
		c.Sendln(fmt.Sprintf("dispatchCommand: command missing handler: [%s]", lookupPath))
		return
	}

	node.Handler(ctx, node, lookupPath, c)
}
