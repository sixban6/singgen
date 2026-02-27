package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"time"
)

// TestEvent represents a single event emitted by "go test -json".
type TestEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Elapsed float64   `json:"Elapsed"` // seconds
	Output  string    `json:"Output"`
}

// TestCase represents the final result of a single test function
// within a package.
type TestCase struct {
	Name    string
	Status  string // pass, fail, skip
	Elapsed float64
	Output  []string // Captured standard output or test logs
}

// TestPackage represents the results of an entire package.
type TestPackage struct {
	Name    string
	Status  string
	Elapsed float64
	Tests   []*TestCase
	Output  []string
}

const htmlTemplateStr = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hysteria2 Bandwidth Testing Report</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 20px; background-color: #f8f9fa; color: #333; }
        h1 { color: #2c3e50; border-bottom: 2px solid #3498db; padding-bottom: 10px; }
        .summary { background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 20px; }
        .summary-stats { display: flex; gap: 20px; margin-top: 15px; }
        .stat-box { padding: 15px; border-radius: 6px; text-align: center; flex: 1; color: white; }
        .stat-total { background-color: #34495e; }
        .stat-pass { background-color: #2ecc71; }
        .stat-fail { background-color: #e74c3c; }
        .stat-skip { background-color: #f1c40f; color: #333; }
        .pkg { background: #fff; margin-bottom: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; }
        .pkg-header { padding: 15px 20px; background-color: #ecf0f1; cursor: pointer; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #ddd; }
        .pkg-header:hover { background-color: #e0e6ed; }
        .pkg-name { font-size: 1.2em; font-weight: bold; }
        .badge { padding: 5px 10px; border-radius: 4px; font-weight: bold; font-size: 0.9em; }
        .pass { background-color: #d4edda; color: #155724; }
        .fail { background-color: #f8d7da; color: #721c24; }
        .skip { background-color: #fff3cd; color: #856404; }
        .test-list { list-style: none; padding: 0; margin: 0; }
        .test-item { padding: 15px 20px; border-bottom: 1px solid #eee; }
        .test-item:last-child { border-bottom: none; }
        .test-header { display: flex; justify-content: space-between; align-items: center; cursor: pointer; }
        .test-name { font-family: monospace; font-size: 1.1em; }
        .test-output { background-color: #272822; color: #f8f8f2; padding: 15px; border-radius: 4px; margin-top: 10px; font-family: monospace; white-space: pre-wrap; font-size: 0.9em; display: none; }
        .expand-icon { font-size: 0.8em; color: #7f8c8d; }
    </style>
</head>
<body>

    <h1>🚀 Hysteria2 Bandwidth Testing Report</h1>

    <div class="summary">
        <h2>Summary</h2>
        <p><strong>Generated at:</strong> {{ .ReportTime }}</p>
        <div class="summary-stats">
            <div class="stat-box stat-total"><h3>Total Tests</h3><h2>{{ .TotalTests }}</h2></div>
            <div class="stat-box stat-pass"><h3>Passed</h3><h2>{{ .TotalPass }}</h2></div>
            <div class="stat-box stat-fail"><h3>Failed</h3><h2>{{ .TotalFail }}</h2></div>
            <div class="stat-box stat-skip"><h3>Skipped</h3><h2>{{ .TotalSkip }}</h2></div>
        </div>
    </div>

    {{ range .Packages }}
    <div class="pkg">
        <div class="pkg-header" onclick="toggleVisibility('pkg-{{.Name}}')">
            <span class="pkg-name">📦 {{ .Name }} ({{ .Elapsed }}s)</span>
            <span class="badge {{ .Status }}">{{ .Status }}</span>
        </div>
        <div id="pkg-{{.Name}}" style="display: block;">
            <ul class="test-list">
                {{ range .Tests }}
                <li class="test-item">
                    <div class="test-header" onclick="toggleVisibility('test-{{.Name}}')">
                        <span class="test-name">
                            <span class="badge {{ .Status }}">{{ .Status }}</span> 
                            {{ .Name }}
                        </span>
                        <span class="expand-icon">⏱️ {{ .Elapsed }}s 🔽</span>
                    </div>
                    <div id="test-{{.Name}}" class="test-output">{{ range .Output }}{{ . }}{{ end }}</div>
                </li>
                {{ end }}
            </ul>
            {{ if .Output }}
            <div style="padding: 15px; font-weight: bold; cursor: pointer;" onclick="toggleVisibility('pkg-output-{{.Name}}')">
                Package Output 🔽
            </div>
            <div id="pkg-output-{{.Name}}" class="test-output" style="margin: 0 20px 20px 20px;">{{ range .Output }}{{ . }}{{ end }}</div>
            {{ end }}
        </div>
    </div>
    {{ end }}

    <script>
        function toggleVisibility(id) {
            var el = document.getElementById(id);
            if (el.style.display === "none" || el.style.display === "") {
                el.style.display = "block";
            } else {
                el.style.display = "none";
            }
        }
    </script>
</body>
</html>
`

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: generate_report <input-json-file> <output-html-file>")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("Error opening input file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	packages := make(map[string]*TestPackage)
	var activeTests = make(map[string]*TestCase) // map "package.TestName" to TestCase

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var event TestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip non-JSON lines or parse errors
			continue
		}

		// Ensure package exists
		pkg, ok := packages[event.Package]
		if !ok {
			pkg = &TestPackage{
				Name:  event.Package,
				Tests: []*TestCase{},
			}
			packages[event.Package] = pkg
		}

		if event.Test == "" {
			// Package level event
			pkg.Output = append(pkg.Output, event.Output)
			if event.Action == "pass" || event.Action == "fail" || event.Action == "skip" {
				pkg.Status = event.Action
				pkg.Elapsed = event.Elapsed
			}
		} else {
			// Test level event
			testKey := event.Package + "." + event.Test
			test, tok := activeTests[testKey]
			if !tok {
				test = &TestCase{
					Name:   event.Test,
					Output: []string{},
				}
				activeTests[testKey] = test
				// Add to package
				pkg.Tests = append(pkg.Tests, test)
			}

			if event.Output != "" {
				test.Output = append(test.Output, event.Output)
			}

			if event.Action == "pass" || event.Action == "fail" || event.Action == "skip" {
				test.Status = event.Action
				test.Elapsed = event.Elapsed
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		os.Exit(1)
	}

	// Prepare data for template
	var reportData struct {
		ReportTime string
		TotalTests int
		TotalPass  int
		TotalFail  int
		TotalSkip  int
		Packages   []*TestPackage
	}

	reportData.ReportTime = time.Now().Format("2006-01-02 15:04:05")

	for _, pkg := range packages {
		if pkg.Status == "" {
			pkg.Status = "skip" // Fallback
		}
		// Sort tests? Or just leave as received. Let's leave as is.
		reportData.Packages = append(reportData.Packages, pkg)
		for _, test := range pkg.Tests {
			// Only count leaf tests (no subtests for simplicity, or count all)
			if test.Status != "" {
				reportData.TotalTests++
				switch test.Status {
				case "pass":
					reportData.TotalPass++
				case "fail":
					reportData.TotalFail++
				case "skip":
					reportData.TotalSkip++
				}
			}
		}
	}

	// Generate HTML
	t, err := template.New("report").Parse(htmlTemplateStr)
	if err != nil {
		fmt.Printf("Error compiling template: %v\n", err)
		os.Exit(1)
	}

	out, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	if err := t.Execute(out, reportData); err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated report at: %s\n", outputFile)
}
