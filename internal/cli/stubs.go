package cli

import "fmt"

func RunPing(_ []string) int       { fmt.Println("not implemented"); return ExitUsage }
func RunCompanies(_ []string) int  { return RunPing(nil) }
func RunObject(_ []string) int     { return RunPing(nil) }
func RunCollection(_ []string) int { return RunPing(nil) }
func RunReport(_ []string) int     { return RunPing(nil) }
func RunRaw(_ []string) int        { return RunPing(nil) }
