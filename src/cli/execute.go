package cli

import (
	"fmt"
	"log"
	//"strings"

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

	commandFeedback(c, getHostname(ctx.ConfRootActive()))
}

func getHostname(root *command.ConfNode) string {
	node, err := root.Get("hostname")
	if err != nil {
		return "hostname?"
	}

	return node.Value[0]
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
		//if err := command.dispatchCommand(ctx, line, c, status); err != nil {
		if err := command.Dispatch(ctx, line, c, status); err != nil {
			c.Sendln(fmt.Sprintf("executeLine: error: %v", err))
		}
	default:
		msg := fmt.Sprintf("unknown state for command: [%s]", line)
		log.Print(msg)
		c.Sendln(msg)
	}

	commandFeedback(c, getHostname(ctx.ConfRootActive()))
}

func commandFeedback(c *Client, hostname string) {
	paging := c.SendQueue()
	c.SetSendEveryChar(paging)
	c.SendPrompt(hostname, paging)
	c.Flush()
}

/*
func dispatchCommand(ctx command.ConfContext, rawLine string, c command.CmdClient, status int) error {

	line := strings.TrimLeft(rawLine, " ")

	if line == "" || line[0] == '!' || line[0] == '#' {
		return nil // ignore empty lines
	}

	c.HistoryAdd(rawLine)

	node, lookupPath, err := command.CmdFindRelative(ctx.CmdRoot(), line, c.ConfigPath(), status)
	if err != nil {
		e := fmt.Errorf("dispatchCommand: not found [%s]: %v", line, err)
		return e
	}

	if node.Handler == nil {
		if node.IsConfig() {
			c.ConfigPathSet(lookupPath) // enter config path
			return nil
		}
		err := fmt.Errorf("dispatchCommand: command missing handler: [%s]", line)
		return err
	}

	node.Handler(ctx, node, lookupPath, c)

	return nil
}
*/
