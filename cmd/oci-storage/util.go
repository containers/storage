package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func jsonEncodeToStdout(item interface{}) int {
	if err := json.NewEncoder(os.Stdout).Encode(container); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}
