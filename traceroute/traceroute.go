package traceroute

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
)

type TracerouteResults []TracerouteResult

type TracerouteResult struct {
	Hop         string
	Destination string
	DNSName     string
	IP          string
	RTT1        string
	RTT2        string
	RTT3        string
}

func RunTraceroute(destination net.IP) (string, error) {
	if destination == nil {
		return "", fmt.Errorf("Destination is not a valid IP.")
	}
	logrus.Infof("Initiating traceroute to %s", destination.String())
	cmd := exec.Command("traceroute", "-m", "15", destination.String())
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	logrus.Infof("Traceroute to %s complete", destination.String())
	return out.String(), nil
}

func ProcessTracerouteOutput(result string) (TracerouteResults, error) {
	var processedResults TracerouteResults
	resultLines := strings.Split(result, "\n")
	for _, line := range resultLines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}
		splitLine := strings.Split(trimmedLine, "  ")
		var processedResult TracerouteResult
		for i, item := range splitLine {
			switch i {
			case 0:
				processedResult.Hop = strings.TrimSpace(item)
			case 1:
				processedResult.Destination = strings.TrimSpace(item)
			case 2:
				processedResult.RTT1 = strings.TrimSpace(item)
			case 3:
				processedResult.RTT2 = strings.TrimSpace(item)
			case 4:
				processedResult.RTT3 = strings.TrimSpace(item)
			}
		}
		processedResults = append(processedResults, processedResult)
	}

	processedResults, err := processDestinationToDNSAndIP(processedResults)
	if err != nil {
		return processedResults, err
	}

	return processedResults, nil
}

func processDestinationToDNSAndIP(processedResults TracerouteResults) (TracerouteResults, error) {
	hostAndIPRegexp, err := regexp.Compile(`(?P<dns>[\w-\.]+) \((?P<ip>\d+\.\d+\.\d+\.\d+)\)`)
	if err != nil {
		return processedResults, err
	}

	var furtherProcessedResults TracerouteResults
	for _, result := range processedResults {
		match := hostAndIPRegexp.FindStringSubmatch(result.Destination)
		if match != nil {
			for i, name := range hostAndIPRegexp.SubexpNames() {
				if i > 0 && i <= len(match) {
					if name == "dns" {
						result.DNSName = match[i]
					} else if name == "ip" {
						result.IP = match[i]
					}
				}
			}
		}

		furtherProcessedResults = append(furtherProcessedResults, result)
	}

	return furtherProcessedResults, nil
}
