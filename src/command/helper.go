package command

import (
	"log"
)

func HelperHostname(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	line, host := StripLastToken(line)

	path, _ := StripLastToken(node.Path)

	confCand := ctx.ConfRootCandidate()
	confNode, err, _ := confCand.Set(path, line)
	if err != nil {
		log.Printf("hostname: error: %v", err)
		return
	}

	confNode.ValueSet(host)
}
