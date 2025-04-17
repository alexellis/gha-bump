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
	var verbose bool

	flag.BoolVar(&write, "write", false, "Write changes to the file")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")

	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "gha-bump ./workflow.yaml\n")
		os.Exit(1)
	}

	target := flag.Args()[0]
	if err := run(write, verbose, target); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func run(write, verbose bool, target string) error {
	if _, err := os.Stat(target); err != nil {
		return err
	}

	data, err := os.ReadFile(target)
	if err != nil {
		return err
	}

	workflow, err := loadWorkflow(data)
	if err != nil {
		return err
	}

	clientWithoutRedirects := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	replacements, err := processJobs(workflow, clientWithoutRedirects, verbose)
	if err != nil {
		return err
	}

	if verbose && len(replacements) > 0 {
		fmt.Println("Detected following replacements:- ")
		for old, new := range replacements {
			fmt.Printf("  %s -> %s\n", old, new)
		}
	}

	if write {
		if err := applyReplacements(data, replacements, target); err != nil {
			return err
		}
	}

	return nil
}

func loadWorkflow(data []byte) (map[string]interface{}, error) {
	return parseWorkflow(data)
}

func processJobs(workflow map[string]interface{}, client *http.Client, verbose bool) (map[string]string, error) {
	jobs, ok := workflow["jobs"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("jobs not found in workflow")
	}

	replace := map[string]string{}

	for jobName, job := range jobs {
		jobReplacements, err := processJob(jobName, job, client, verbose)
		if err != nil {
			return nil, err
		}
		for k, v := range jobReplacements {
			replace[k] = v
		}
	}

	return replace, nil
}

func processJob(jobName string, job interface{}, client *http.Client, verbose bool) (map[string]string, error) {
	jobMap, ok := job.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("job %s is not a map", jobName)
	}

	steps, ok := jobMap["steps"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("steps not found or not a list in job %s", jobName)
	}

	replacements := map[string]string{}

	for _, step := range steps {
		stepMap, ok := step.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("step is not a map in job %s", jobName)
		}

		var name string
		var uses string

		if stepMap["name"] != nil {
			if v, ok := stepMap["name"].(string); ok {
				name = v
			}
		}
		if stepMap["uses"] != nil {
			if v, ok := stepMap["uses"].(string); ok {
				uses = v
			}
		}

		st := name
		if name == "" {
			st = uses
		}

		if verbose {
			if len(uses) > 0 {
				fmt.Printf("  %s: %s\n", st, uses)
			} else {
				fmt.Printf("  %s\n", st)
			}
		}

		if len(uses) > 0 {
			if newVer, err := suggestMajorUpgrade(client, uses); err != nil {
				return nil, err
			} else if newVer != "" {
				replacements[uses] = newVer
			}
		}
	}

	return replacements, nil
}

func suggestMajorUpgrade(client *http.Client, uses string) (string, error) {
	ownerRepo, currentVer, ok := strings.Cut(uses, "@")
	if !ok || currentVer == "master" {
		return "", nil
	}
	owner, repo, ok := strings.Cut(ownerRepo, "/")
	if !ok {
		return "", nil
	}
	version, err := getLatestVersion(client, owner, repo)
	if err != nil {
		return "", err
	}
	oldSemver, err := semver.NewVersion(currentVer)
	if err != nil {
		return "", err
	}
	newSemver, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}
	if newSemver.Major() > oldSemver.Major() {
		return fmt.Sprintf("v%d", newSemver.Major()), nil
	}
	return "", nil
}

func applyReplacements(data []byte, replacements map[string]string, target string) error {
	if len(replacements) == 0 {
		return nil
	}
	dataSt := string(data)
	for old, new := range replacements {
		oldFull := old
		workflowPath, _, _ := strings.Cut(oldFull, "@")
		newFull := fmt.Sprintf("%s@%s", workflowPath, new)
		dataSt = strings.ReplaceAll(dataSt, oldFull, newFull)
	}

	if err := os.WriteFile(target, []byte(dataSt), 0644); err != nil {
		return err
	}

	return nil
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
