package command

import (
	"fmt"
	"log"
)

type commitAction struct {
	cmd    string
	enable bool
}

func ConfEqual(root1, root2 *ConfNode) bool {
	return len(findDeleted(root1, root2))+len(findDeleted(root2, root1)) == 0
}

// get diff from active conf to candidate conf
// build command list to apply diff to active conf
//  - include preparatory commands, like deleting addresses from interfaces affected by address change
//  - if any command fails, revert previously applied commands
func Commit(ctx ConfContext, c CmdClient) error {
	confAct := ctx.ConfRootActive()
	confCand := ctx.ConfRootCandidate()

	c.Sendln("commit: building commit plan")

	var commitPlan []commitAction

	//c.Sendln("deleted from active to candidate:")
	disableList := findDeleted(confAct, confCand)
	for _, conf := range disableList {
		//c.Sendln(fmt.Sprintf("commit: %s", conf))
		commitPlan = append(commitPlan, commitAction{cmd: conf, enable: false})
	}

	//c.Sendln("deleted from candidate to active:")
	enableList := findDeleted(confCand, confAct)
	for _, conf := range enableList {
		//c.Sendln(fmt.Sprintf("commit: %s", conf))
		commitPlan = append(commitPlan, commitAction{cmd: conf, enable: true})
	}

	c.Sendln("commit: applying")

	for i, action := range commitPlan {
		node, err := CmdFind(ctx.CmdRoot(), action.cmd, c.Status())
		if err != nil {
			fail := fmt.Sprintf("Commit: action failed: cmd=[%s] enable=%v: error: %v", action.cmd, action.enable, err)
			log.Printf(fail)
			c.Sendln(fail)
			rollback(ctx, c, commitPlan, i-1)
			return fmt.Errorf(fail)
		}

		continue //FIXME

		if node.Apply == nil {
			fail := fmt.Sprintf("Commit: action failed: cmd=[%s] enable=%v: missing commit func", action.cmd, action.enable)
			log.Printf(fail)
			c.Sendln(fail)
			rollback(ctx, c, commitPlan, i-1)
			return fmt.Errorf(fail)
		}

		if e := node.Apply(ctx, node, action.enable, c); e != nil {
			fail := fmt.Sprintf("Commit: action failed: cmd=[%s] enable=%v: commit func error: %v", action.cmd, action.enable, e)
			log.Printf(fail)
			c.Sendln(fail)
			rollback(ctx, c, commitPlan, i-1)
			return fmt.Errorf(fail)
		}
	}

	c.Sendln("commit: done")

	return nil
}

func rollback(ctx ConfContext, c CmdClient, plan []commitAction, index int) {
}
