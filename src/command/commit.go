package command

import (
	"fmt"
	//"log"
)

// get diff from active conf to candidate conf
// build command list to apply diff to active conf
//  - include preparatory commands, like deleting addresses from interfaces affected by address change
//  - if any command fails, revert previously applied commands
func Commit(ctx ConfContext, c CmdClient) error {
	confAct := ctx.ConfRootActive()
	confCand := ctx.ConfRootCandidate()

	c.Sendln("deleted from active to candidate:")
	cmdList1 := findDeleted(confAct, confCand)
	for _, conf := range cmdList1 {
		c.Sendln(fmt.Sprintf("commit: %s", conf))
	}

	c.Sendln("deleted from candidate to active:")
	cmdList2 := findDeleted(confCand, confAct)
	for _, conf := range cmdList2 {
		c.Sendln(fmt.Sprintf("commit: %s", conf))
	}

	return nil
}
