package cli

import (
	"bufio"
	"fmt"
	"os"

	"command"
)

func LoadConfig(ctx command.ConfContext, path string, c command.CmdClient) error {

	f, err1 := os.Open(path)
	if err1 != nil {
		return fmt.Errorf("cli.LoadConfig: error opening config file: [%s]: %v", path, err1)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if err := dispatchCommand(ctx, line, c, command.CONF); err != nil {
			return fmt.Errorf("cli.LoadConfig: error reading config file: [%s]: %v", path, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("cli.LoadConfig: error scanning config file: [%s]: %v", path, err)
	}

	return nil
}
