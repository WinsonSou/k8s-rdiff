package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Dialog handles user interaction for the diff process
type Dialog struct {
	reader *bufio.Reader
}

// NewDialog creates a new dialog instance
func NewDialog() *Dialog {
	return &Dialog{
		reader: bufio.NewReader(os.Stdin),
	}
}

// WaitForUserAction prompts the user to execute their commands and continue
func (d *Dialog) WaitForUserAction() error {
	fmt.Println("Execute your command(s) and type 'continue' when done.")
	
	for {
		fmt.Print("> ")
		input, err := d.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %v", err)
		}
		
		input = strings.TrimSpace(input)
		if strings.ToLower(input) == "continue" {
			break
		} else if strings.ToLower(input) == "exit" || strings.ToLower(input) == "quit" {
			return fmt.Errorf("diff process canceled by user")
		} else if input != "" {
			fmt.Println("Type 'continue' to proceed with diff, or 'exit' to cancel.")
		}
	}
	
	return nil
}
