package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	coreURL := flag.String("core", "http://127.0.0.1:8090", "control API base URL")
	logPath := flag.String("log", "logs/auxitalkd.log", "core log file path")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println("auxitalkctl 0.1.0-dev")
		fmt.Println("commands:")
		fmt.Println("  status            Show full core status")
		fmt.Println("  plugins           List plugins")
		fmt.Println("  actions           List actions")
		fmt.Println("  workflows         List workflows")
		fmt.Println("  logs              Print core log file")
		fmt.Println("  approve <id>      Approve an action by ID")
		fmt.Println("  deny <id>         Deny an action by ID")
		return
	}

	cmd := flag.Arg(0)
	switch cmd {
	case "status":
		printStatus(*coreURL)
	case "plugins":
		printPlugins(*coreURL)
	case "actions":
		printActions(*coreURL)
	case "workflows":
		printWorkflows(*coreURL)
	case "logs":
		printLogs(*logPath)
	case "approve":
		if flag.NArg() < 2 {
			fmt.Println("missing action id")
			os.Exit(1)
		}
		mutateAction(*coreURL, flag.Arg(1), "approve")
	case "deny":
		if flag.NArg() < 2 {
			fmt.Println("missing action id")
			os.Exit(1)
		}
		mutateAction(*coreURL, flag.Arg(1), "deny")
	default:
		fmt.Printf("unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func printStatus(base string) {
	data := fetch(base + "/api/status")
	fmt.Printf("%s\n", pretty(data))
}

func printPlugins(base string) {
	data := fetch(base + "/api/plugins")
	fmt.Printf("%s\n", pretty(data))
}

func printActions(base string) {
	data := fetch(base + "/api/actions")
	fmt.Printf("%s\n", pretty(data))
}

func printWorkflows(base string) {
	data := fetch(base + "/api/workflows")
	fmt.Printf("%s\n", pretty(data))
}

func printLogs(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("error reading logs: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(string(data))
}

func mutateAction(base, id, operation string) {
	url := fmt.Sprintf("%s/api/actions/%s/%s", base, id, operation)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		fmt.Printf("error building request: %v\n", err)
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		fmt.Printf("error: server returned %s\n", resp.Status)
		os.Exit(1)
	}
	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	_ = json.Unmarshal(body, &result)
	fmt.Printf("%s action %s:\n%s\n", operation, id, pretty(result))
}

func fetch(url string) any {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result any
	_ = json.Unmarshal(body, &result)
	return result
}

func pretty(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
