package cli

import "fmt"

func RunPing(_ []string) int       { fmt.Println("not implemented"); return ExitUsage }
func RunCompanies(_ []string) int  { return RunPing(nil) }
