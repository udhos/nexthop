package cli

import (
	"fmt"
	"log"
	//"strings"

	"command"
)

func Execute(ctx command.ConfContext, line string, isLine, history bool, c *Client) {
	log.Printf("cli.Execute: isLine=%v cmd=[%s]", isLine, line)

	if isLine {
		// full-line command
		executeLine(ctx, line, history, c)
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

	return command.LastToken(node.Children[0].Path)
}

func isAnyUserDefined(root *command.ConfNode) bool {
	node, err := root.Get("username")
	if err != nil {
		return false
	}

	return len(node.Children) > 0
}

func checkPassword(root *command.ConfNode, username, password string) bool {
	path := fmt.Sprintf("username %s password", username)

	node, err := root.Get(path)
	if err != nil {
		return false
	}

	if len(node.Children) != 1 {
		return false
	}

	return password == command.LastToken(node.Children[0].Path)
}

func executeLine(ctx command.ConfContext, line string, history bool, c *Client) {
	log.Printf("executeLine: [%v]", line)

	status := c.Status()

	switch status {
	case command.MOTD:
		c.Sendln("")
		c.Sendln("vty server ready")
		c.Sendln("")
		if isAnyUserDefined(ctx.ConfRootActive()) {
			c.StatusSet(command.USER) // request user/password auth
		} else {
			c.StatusSet(command.EXEC) // login without user/password auth
		}
	case command.USER:
		c.EchoDisable()
		c.StatusSet(command.PASS)
		c.UsernameSet(line)
	case command.PASS:
		c.EchoEnable()
		if checkPassword(ctx.ConfRootActive(), c.Username(), line) {
			c.StatusSet(command.EXEC) // login allowed
		} else {
			c.StatusSet(command.USER) // request user again
		}
	case command.EXEC, command.ENAB, command.CONF:
		if err := command.Dispatch(ctx, line, c, status, history); err != nil {
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
