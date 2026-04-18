package main

import doctheaterlib "github.com/Bucket-Chemist/goYoke/internal/hooks/doctheater"

func escapeJSON(s string) string          { return doctheaterlib.EscapeJSON(s) }
func allowResponse() string               { return doctheaterlib.AllowResponse() }
func warnResponse(message string) string  { return doctheaterlib.WarnResponse(message) }
func blockResponse(message string) string { return doctheaterlib.BlockResponse(message) }
func outputError(message string)          { doctheaterlib.OutputError(message) }

func main() { doctheaterlib.Main() }
