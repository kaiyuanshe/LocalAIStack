package system

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type BaseInfoSummary struct {
	CPUCores int
	MemoryKB int64
	GPUName  string
}

func LoadBaseInfoSummary(path string) (BaseInfoSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return BaseInfoSummary{}, err
	}
	defer file.Close()

	var summary BaseInfoSummary
	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "### ") {
			section = strings.TrimSpace(strings.TrimPrefix(line, "### "))
			continue
		}
		if line == "" {
			continue
		}

		switch {
		case section == "CPU" && strings.HasPrefix(line, "- Cores:"):
			value := strings.TrimSpace(strings.TrimPrefix(line, "- Cores:"))
			if cores, err := strconv.Atoi(value); err == nil {
				summary.CPUCores = cores
			}
		case section == "Memory" && strings.HasPrefix(line, "- Total:"):
			value := strings.TrimSpace(strings.TrimPrefix(line, "- Total:"))
			if fields := strings.Fields(value); len(fields) >= 2 {
				if total, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
					summary.MemoryKB = total
				}
			}
		case section == "GPU" && strings.HasPrefix(line, "- GPU:") && summary.GPUName == "":
			summary.GPUName = strings.TrimSpace(strings.TrimPrefix(line, "- GPU:"))
		}
	}

	if err := scanner.Err(); err != nil {
		return BaseInfoSummary{}, fmt.Errorf("read base info: %w", err)
	}

	return summary, nil
}
