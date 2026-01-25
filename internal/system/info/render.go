package info

import (
	"fmt"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

func RenderBaseInfoMarkdown(info BaseInfo, rawOutputs []RawCommandOutput) string {
	var builder strings.Builder
	builder.WriteString(i18n.T("# LocalAIStack Base Info\n\n"))
	builder.WriteString(i18n.T("- Timestamp: %s\n\n", info.CollectedAt))

	builder.WriteString(i18n.T("## System Information\n\n"))
	builder.WriteString(i18n.T("### OS\n"))
	builder.WriteString(i18n.T("- OS: %s\n- Arch: %s\n\n", info.OS, info.Arch))

	builder.WriteString(i18n.T("### Kernel\n"))
	builder.WriteString(i18n.T("- Kernel: %s\n\n", info.Kernel))

	builder.WriteString(i18n.T("### CPU\n"))
	builder.WriteString(i18n.T("- Model: %s\n- Cores: %d\n\n", info.CPUModel, info.CPUCores))

	builder.WriteString(i18n.T("### GPU\n"))
	builder.WriteString(i18n.T("- GPU: %s\n\n", info.GPU))

	builder.WriteString(i18n.T("### Memory\n"))
	builder.WriteString(i18n.T("- Total: %s\n\n", info.MemoryTotal))

	builder.WriteString(i18n.T("### Disk\n"))
	builder.WriteString(i18n.T("- Total: %s\n- Available: %s\n\n", info.DiskTotal, info.DiskAvailable))

	builder.WriteString(i18n.T("### Network\n"))
	builder.WriteString(i18n.T("- Hostname: %s\n- Internal IPs: %s\n\n", info.Hostname, formatStringList(info.InternalIPs)))

	builder.WriteString(i18n.T("### Runtime\n"))
	builder.WriteString(i18n.T("- Docker: %s\n- Podman: %s\n- Capabilities: %s\n\n", info.Docker, info.Podman, info.RuntimeCapabilities))

	builder.WriteString(i18n.T("### Version\n"))
	builder.WriteString(i18n.T("- LocalAIStack: %s\n\n", info.LocalAIStackVersion))

	builder.WriteString(i18n.T("## Raw Command Outputs\n\n"))
	for _, raw := range rawOutputs {
		builder.WriteString("<details>\n")
		builder.WriteString(i18n.T("<summary>%s</summary>\n\n", raw.Command))
		builder.WriteString(i18n.T("```text\n"))
		builder.WriteString(formatRawOutput(raw))
		builder.WriteString(i18n.T("\n```\n\n"))
		builder.WriteString("</details>\n\n")
	}

	return builder.String()
}

func formatRawOutput(raw RawCommandOutput) string {
	var parts []string
	if raw.Err != "" {
		parts = append(parts, i18n.T("error: %s", raw.Err))
	}
	if strings.TrimSpace(raw.Stdout) != "" {
		parts = append(parts, i18n.T("stdout:"))
		parts = append(parts, strings.TrimRight(raw.Stdout, "\n"))
	}
	if strings.TrimSpace(raw.Stderr) != "" {
		parts = append(parts, i18n.T("stderr:"))
		parts = append(parts, strings.TrimRight(raw.Stderr, "\n"))
	}
	if len(parts) == 0 {
		return i18n.T("no output")
	}
	return strings.Join(parts, "\n")
}

func formatStringList(values []string) string {
	if len(values) == 0 {
		return i18n.T("unknown")
	}
	return strings.Join(values, ", ")
}
