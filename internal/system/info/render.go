package info

import (
	"fmt"
	"strings"
)

func RenderBaseInfoMarkdown(info BaseInfo, rawOutputs []RawCommandOutput) string {
	var builder strings.Builder
	builder.WriteString("# LocalAIStack Base Info\n\n")
	builder.WriteString(fmt.Sprintf("- Timestamp: %s\n\n", info.CollectedAt))

	builder.WriteString("## System Information\n\n")
	builder.WriteString("### OS\n")
	builder.WriteString(fmt.Sprintf("- OS: %s\n- Arch: %s\n\n", info.OS, info.Arch))

	builder.WriteString("### Kernel\n")
	builder.WriteString(fmt.Sprintf("- Kernel: %s\n\n", info.Kernel))

	builder.WriteString("### CPU\n")
	builder.WriteString(fmt.Sprintf("- Model: %s\n- Cores: %d\n\n", info.CPUModel, info.CPUCores))

	builder.WriteString("### GPU\n")
	builder.WriteString(fmt.Sprintf("- GPU: %s\n\n", info.GPU))

	builder.WriteString("### Memory\n")
	builder.WriteString(fmt.Sprintf("- Total: %s\n\n", info.MemoryTotal))

	builder.WriteString("### Disk\n")
	builder.WriteString(fmt.Sprintf("- Total: %s\n- Available: %s\n\n", info.DiskTotal, info.DiskAvailable))

	builder.WriteString("### Network\n")
	builder.WriteString(fmt.Sprintf("- Hostname: %s\n- Internal IPs: %s\n\n", info.Hostname, formatStringList(info.InternalIPs)))

	builder.WriteString("### Runtime\n")
	builder.WriteString(fmt.Sprintf("- Docker: %s\n- Podman: %s\n- Capabilities: %s\n\n", info.Docker, info.Podman, info.RuntimeCapabilities))

	builder.WriteString("### Version\n")
	builder.WriteString(fmt.Sprintf("- LocalAIStack: %s\n\n", info.LocalAIStackVersion))

	builder.WriteString("## Raw Command Outputs\n\n")
	for _, raw := range rawOutputs {
		builder.WriteString("<details>\n")
		builder.WriteString(fmt.Sprintf("<summary>%s</summary>\n\n", raw.Command))
		builder.WriteString("```text\n")
		builder.WriteString(formatRawOutput(raw))
		builder.WriteString("\n```\n\n")
		builder.WriteString("</details>\n\n")
	}

	return builder.String()
}

func formatRawOutput(raw RawCommandOutput) string {
	var parts []string
	if raw.Err != "" {
		parts = append(parts, fmt.Sprintf("error: %s", raw.Err))
	}
	if strings.TrimSpace(raw.Stdout) != "" {
		parts = append(parts, "stdout:")
		parts = append(parts, strings.TrimRight(raw.Stdout, "\n"))
	}
	if strings.TrimSpace(raw.Stderr) != "" {
		parts = append(parts, "stderr:")
		parts = append(parts, strings.TrimRight(raw.Stderr, "\n"))
	}
	if len(parts) == 0 {
		return "no output"
	}
	return strings.Join(parts, "\n")
}

func formatStringList(values []string) string {
	if len(values) == 0 {
		return "unknown"
	}
	return strings.Join(values, ", ")
}
