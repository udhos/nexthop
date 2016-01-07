package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"command"
)

func LoadConfig(ctx command.ConfContext, path string, c command.CmdClient, abortOnError bool) error {

	f, err1 := os.Open(path)
	if err1 != nil {
		return fmt.Errorf("cli.LoadConfig: error opening config file: [%s]: %v", path, err1)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)

	var lastErr error

	for scanner.Scan() {
		line := scanner.Text()
		if err := dispatchCommand(ctx, line, c, command.CONF); err != nil {
			lastErr = fmt.Errorf("cli.LoadConfig: error reading config file: [%s]: %v", path, err)
			log.Printf("%v", lastErr)
			if abortOnError {
				return lastErr
			}
		}
	}

	if err := scanner.Err(); err != nil {
		lastErr = fmt.Errorf("cli.LoadConfig: error scanning config file: [%s]: %v", path, err)
	}

	return lastErr
}
