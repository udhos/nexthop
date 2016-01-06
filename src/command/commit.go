package command

import (
	"fmt"
	"log"
)

type CommitAction struct {
	Cmd    string
	Enable bool
	conf   *ConfNode
}

func ConfEqual(root1, root2 *ConfNode) bool {
	pathList1, _ := findDeleted(root1, root2)
	pathList2, _ := findDeleted(root2, root1)
	return len(pathList1)+len(pathList2) == 0
}

// get diff from active conf to candidate conf
// build command list to apply diff to active conf
//  - include preparatory commands, like deleting addresses from interfaces affected by address change
//  - if any command fails, revert previously applied commands
func Commit(ctx ConfContext, c CmdClient, forceFailure bool) error {
	confAct := ctx.ConfRootActive()
	confCand := ctx.ConfRootCandidate()

	c.Sendln("commit: building commit plan")

	var commitPlan []CommitAction

	//c.Sendln("deleted from active to candidate:")
	disablePathList, disableNodeList := findDeleted(confAct, confCand)
	for i, cmdPath := range disablePathList {
		commitPlan = append(commitPlan, CommitAction{Cmd: cmdPath, conf: disableNodeList[i], Enable: false})
	}

	//c.Sendln("deleted from candidate to active:")
	enablePathList, enableNodeList := findDeleted(confCand, confAct)
	for i, cmdPath := range enablePathList {
		commitPlan = append(commitPlan, CommitAction{Cmd: cmdPath, conf: enableNodeList[i], Enable: true})
	}

	c.Sendln("commit: applying")

	for i, action := range commitPlan {
		c.Sendln(fmt.Sprintf("commit: applying: action[%d]: cmd=[%s] enable=%v", i, action.Cmd, action.Enable))
		node, err := CmdFind(ctx.CmdRoot(), action.Cmd, c.Status())
		if err != nil {
			fail := fmt.Sprintf("Commit: action failed: cmd=[%s] enable=%v: error: %v", action.Cmd, action.Enable, err)
			log.Printf(fail)
			c.Sendln(fail)
			revert(ctx, c, commitPlan, i-1)
			return fmt.Errorf(fail)
		}

		// force last action to fail
		if forceFailure && i == len(commitPlan)-1 {
			fail := fmt.Sprintf("Commit: action[%d] failed: cmd=[%s] enable=%v: error: HARD-CODED FAILURE", i, action.Cmd, action.Enable)
			log.Printf(fail)
			c.Sendln(fail)
			revert(ctx, c, commitPlan, i-1)
			return fmt.Errorf(fail)
		}

		if node.Apply == nil {
			fail := fmt.Sprintf("Commit: action[%d] failed: cmd=[%s] enable=%v: missing commit func", i, action.Cmd, action.Enable)
			log.Printf(fail)
			c.Sendln(fail)
			revert(ctx, c, commitPlan, i-1)
			return fmt.Errorf(fail)
		}

		if e := node.Apply(ctx, node, action, c); e != nil {
			fail := fmt.Sprintf("Commit: action[%d] failed: cmd=[%s] enable=%v: commit func error: %v", i, action.Cmd, action.Enable, e)
			log.Printf(fail)
			c.Sendln(fail)
			revert(ctx, c, commitPlan, i-1)
			return fmt.Errorf(fail)
		}
	}

	c.Sendln("commit: done")

	return nil
}

func revert(ctx ConfContext, c CmdClient, plan []CommitAction, index int) {
	for i := index; i >= 0; i-- {
		action := plan[i]

		action.Enable = !action.Enable

		c.Sendln(fmt.Sprintf("revert: action[%d] cmd=[%s] enable=%v", i, action.Cmd, action.Enable))
		node, err := CmdFind(ctx.CmdRoot(), action.Cmd, c.Status())
		if err != nil {
			fail := fmt.Sprintf("revert: action[%d] failed: cmd=[%s] enable=%v: error: %v", i, action.Cmd, action.Enable, err)
			log.Printf(fail)
			c.Sendln(fail)
			continue
		}

		if node.Apply == nil {
			fail := fmt.Sprintf("revert: action[%d] failed: cmd=[%s] enable=%v: missing commit func", i, action.Cmd, action.Enable)
			log.Printf(fail)
			c.Sendln(fail)
			continue
		}

		if e := node.Apply(ctx, node, action, c); e != nil {
			fail := fmt.Sprintf("revert: action[%d] failed: cmd=[%s] enable=%v: commit func error: %v", i, action.Cmd, action.Enable, e)
			log.Printf(fail)
			c.Sendln(fail)
			continue
		}
	}
}
