package util

import (
	"fmt"
	"strings"
)

const (
	// StaragentURL URL of SA
	StaragentServer = "inc.agent.alibaba-inc.com"
	// StaragentAPIKey API key of SA
	StaragentAPIKey = "0eb5fc49b702c8f4ac3d17d5950af8ec"
	// StaragentAPICode API code of SA
	StaragentAPICode = "3a9338cfab4e1462fe51c69f295102f0"
)

var (
	saClient = &SaClient{
		Key:    StaragentAPIKey,
		Sign:   StaragentAPICode,
		Server: StaragentServer,
	}
)

// ResponseFromStarAgentTask run the cmd on specified host and get response
func ResponseFromStarAgentTask(cmd, hostIP, hostSN string) (string, error) {
	if arr := strings.Split(cmd, "cmd://"); len(arr) != 2 {
		return "", fmt.Errorf("cmd: %s, cmd must start with cmd://", cmd)
	} else {
		cmd = arr[1]
	}

	return saClient.Cmd(hostIP, cmd)
}
