package perf

import "fmt"

func BuildPerfScriptCommand(exe, input string) string {
	return fmt.Sprintf("%s script -i %s -F event,period,comm,ip,sym", exe, input)
}
