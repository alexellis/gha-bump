package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
)

func main() {

	var write bool

	if len(os.Args) <= 1 {
		fmt.Fprintf(os.Stderr, "gha-bump ./workflow.yaml\n")
		os.Exit(1)
	}

	flag.BoolVar(&write, "write", false, "Write changes to the file")

	flag.Parse()

	target := os.Args[1]
	if _, err := os.Stat(target); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Parse the workflow file
	workflow, err := parseWorkflow(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	clientWithoutRedirects := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	fmt.Println("Workflow parsed successfully.")

	jobs, ok := workflow["jobs"].(map[string]interface{})
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: jobs not found in workflow\n")
		os.Exit(1)
	}

	for jobName, job := range jobs {
		jobMap, ok := job.(map[string]interface{})
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: job %s is not a map\n", jobName)
			os.Exit(1)
		}

		steps := jobMap["steps"].([]interface{})

		replace := map[string]string{}

		for _, step := range steps {
			stepMap, ok := step.(map[string]interface{})
			if !ok {
				fmt.Fprintf(os.Stderr, "Error: step is not a map in job %s\n", jobName)
				os.Exit(1)
			}

			var name string
			var uses string

			if stepMap["name"] != nil {
				v, ok := stepMap["name"].(string)
				if ok {
					name = v
				}
			}

			if stepMap["uses"] != nil {
				v, ok := stepMap["uses"].(string)
				if ok {
					uses = v
				}
			}

			st := name
			if name == "" {
				st = uses
			}

			if len(uses) > 0 {
				fmt.Printf("  %s: %s\n", st, uses)
			} else {
				fmt.Printf("  %s\n", st)
			}

			if len(uses) > 0 {

				ownerRepo, currentVer, ok := strings.Cut(uses, "@")
				if ok {
					if currentVer != "master" {

						owner, repo, ok := strings.Cut(ownerRepo, "/")
						if ok {
							version, err := getLatestVersion(clientWithoutRedirects, owner, repo)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Error: %s\n", err)
								os.Exit(1)
							}

							oldSemver, err := semver.NewVersion(currentVer)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Error: %s\n", err)
								os.Exit(1)
							}

							newSemver, err := semver.NewVersion(version)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Error: %s\n", err)
								os.Exit(1)
							}

							if newSemver.Major() > oldSemver.Major() {
								shortVer := fmt.Sprintf("v%d", newSemver.Major())

								replace[uses] = shortVer
							}

						}
					}
				}

			}

		}

		fmt.Println("Detected following replacements:- ")

		dataSt := string(data)

		for old, new := range replace {
			fmt.Printf("  %s -> %s\n", old, new)

			oldFull := old
			workflowPath, _, _ := strings.Cut(oldFull, "@")
			newFull := fmt.Sprintf("%s@%s", workflowPath, new)

			dataSt = strings.ReplaceAll(dataSt, oldFull, newFull)
		}

		if err := os.WriteFile(target, []byte(dataSt), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}

	}

}

func parseWorkflow(data []byte) (map[string]interface{}, error) {
	var wf map[string]interface{}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, err
	}

	return wf, nil

}

func getLatestVersion(client *http.Client, owner, repo string) (string, error) {

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://github.com/%s/%s/releases/latest", owner, repo), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	var body []byte
	if res.Body != nil {
		defer res.Body.Close()
		body, _ = io.ReadAll(res.Body)
	}

	if res.StatusCode != http.StatusFound {
		return "", fmt.Errorf("failed to get latest version: %s, body: %s", res.Status, string(body))
	}

	location := res.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("no location header found")
	}

	parts := strings.Split(location, "/")
	if len(parts) < 7 {
		return "", fmt.Errorf("invalid location header: %s", location)
	}

	latestVersion := parts[len(parts)-1]
	return latestVersion, nil
}
