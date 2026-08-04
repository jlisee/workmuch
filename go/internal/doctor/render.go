package doctor

import (
	"fmt"
	"html"
	"strings"
	"time"
)

func RenderText(report DoctorReport) string {
	var b strings.Builder
	b.WriteString("WorkMuch Doctor\n")
	writeLine(&b, "Selected backend", valueOrUnavailable(report.SelectedBackend))
	writeLine(&b, "Active backend", valueOrUnavailable(report.ActiveBackend))
	if report.BackendError != "" {
		writeLine(&b, "Backend error", report.BackendError)
	}
	writeLine(&b, "Accessibility permission", stringOrUnknown(string(report.Permission.State)))
	if report.Permission.Detail != "" {
		writeLine(&b, "Accessibility detail", report.Permission.Detail)
	}
	if report.Permission.Error != "" {
		writeLine(&b, "Accessibility error", report.Permission.Error)
	}
	writeLine(&b, "Frontmost app", valueOrUnavailable(report.Sample.FrontmostApp))
	writeLine(&b, "Focused window title", windowTitleText(report.Sample))
	writeLine(&b, "Idle seconds", fmt.Sprintf("%.2f", report.Sample.IdleSeconds))
	if report.Sample.Error != "" {
		writeLine(&b, "Sample warning", report.Sample.Error)
	}
	writeLine(&b, "Log directory", logDirectoryText(report.Logs))
	writeLine(&b, "Current work log", valueOrUnavailable(report.Logs.CurrentWorkLog))
	if report.Logs.Error != "" {
		writeLine(&b, "Log directory error", report.Logs.Error)
	}
	writeLine(&b, "LaunchAgent", launchAgentText(report.LaunchAgent))
	writeLine(&b, "Runtime status", runtimeStatusText(report.Runtime))
	if report.Runtime.Path != "" {
		writeLine(&b, "Runtime status file", report.Runtime.Path)
	}
	writeLine(&b, "Runtime selected backend", valueOrUnavailable(report.Runtime.Status.SelectedBackend))
	writeLine(&b, "Runtime active backend", valueOrUnavailable(report.Runtime.Status.ActiveBackend))
	writeLine(&b, "Sample count", fmt.Sprintf("%d", report.Runtime.Status.SampleCount))
	writeLine(&b, "Last started", timeText(report.Runtime.Status.StartedAt))
	writeLine(&b, "Last stopped", timeText(report.Runtime.Status.StoppedAt))
	writeLine(&b, "Last sample", timeText(report.Runtime.Status.LastSampleAt))
	writeLine(&b, "Last successful sample", timeText(report.Runtime.Status.LastSuccessfulSampleAt))
	if report.Runtime.Status.CurrentWorkLogPath != "" {
		writeLine(&b, "Runtime work log", report.Runtime.Status.CurrentWorkLogPath)
	}
	if report.Runtime.Status.LatestWarning != nil {
		writeLine(&b, "Latest warning", eventText(report.Runtime.Status.LatestWarning.At, report.Runtime.Status.LatestWarning.Message))
	}
	if report.Runtime.Status.LatestError != nil {
		writeLine(&b, "Latest error", eventText(report.Runtime.Status.LatestError.At, report.Runtime.Status.LatestError.Message))
	}
	if report.Runtime.Error != "" {
		writeLine(&b, "Runtime status error", report.Runtime.Error)
	}
	return b.String()
}

func RenderHTML(report DoctorReport) string {
	rows := []htmlRow{
		{"Selected backend", valueOrUnavailable(report.SelectedBackend)},
		{"Active backend", valueOrUnavailable(report.ActiveBackend)},
		{"Accessibility permission", stringOrUnknown(string(report.Permission.State))},
		{"Frontmost app", valueOrUnavailable(report.Sample.FrontmostApp)},
		{"Focused window title", windowTitleText(report.Sample)},
		{"Idle seconds", fmt.Sprintf("%.2f", report.Sample.IdleSeconds)},
		{"Log directory", logDirectoryText(report.Logs)},
		{"Current work log", valueOrUnavailable(report.Logs.CurrentWorkLog)},
		{"LaunchAgent", launchAgentText(report.LaunchAgent)},
		{"Runtime status", runtimeStatusText(report.Runtime)},
		{"Runtime status file", valueOrUnavailable(report.Runtime.Path)},
		{"Runtime selected backend", valueOrUnavailable(report.Runtime.Status.SelectedBackend)},
		{"Runtime active backend", valueOrUnavailable(report.Runtime.Status.ActiveBackend)},
		{"Sample count", fmt.Sprintf("%d", report.Runtime.Status.SampleCount)},
		{"Last started", timeText(report.Runtime.Status.StartedAt)},
		{"Last stopped", timeText(report.Runtime.Status.StoppedAt)},
		{"Last sample", timeText(report.Runtime.Status.LastSampleAt)},
		{"Last successful sample", timeText(report.Runtime.Status.LastSuccessfulSampleAt)},
		{"Runtime work log", valueOrUnavailable(report.Runtime.Status.CurrentWorkLogPath)},
	}
	if report.BackendError != "" {
		rows = append(rows, htmlRow{"Backend error", report.BackendError})
	}
	if report.Permission.Detail != "" {
		rows = append(rows, htmlRow{"Accessibility detail", report.Permission.Detail})
	}
	if report.Permission.Error != "" {
		rows = append(rows, htmlRow{"Accessibility error", report.Permission.Error})
	}
	if report.Sample.Error != "" {
		rows = append(rows, htmlRow{"Sample warning", report.Sample.Error})
	}
	if report.Logs.Error != "" {
		rows = append(rows, htmlRow{"Log directory error", report.Logs.Error})
	}
	if report.Runtime.Status.LatestWarning != nil {
		rows = append(rows, htmlRow{"Latest warning", eventText(report.Runtime.Status.LatestWarning.At, report.Runtime.Status.LatestWarning.Message)})
	}
	if report.Runtime.Status.LatestError != nil {
		rows = append(rows, htmlRow{"Latest error", eventText(report.Runtime.Status.LatestError.At, report.Runtime.Status.LatestError.Message)})
	}
	if report.Runtime.Error != "" {
		rows = append(rows, htmlRow{"Runtime status error", report.Runtime.Error})
	}

	var rowHTML strings.Builder
	for _, row := range rows {
		rowHTML.WriteString("<tr><th>")
		rowHTML.WriteString(html.EscapeString(row.label))
		rowHTML.WriteString("</th><td>")
		rowHTML.WriteString(html.EscapeString(row.value))
		rowHTML.WriteString("</td></tr>\n")
	}

	return `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>WorkMuch Status</title>
  <style>
    :root {
      color-scheme: light;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      background: #f7f7f5;
      color: #222;
    }
    body {
      margin: 0;
      padding: 24px;
    }
    main {
      max-width: 900px;
      margin: 0 auto;
    }
    h1 {
      margin: 0 0 16px;
      font-size: 28px;
      font-weight: 650;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      background: #fff;
      border: 1px solid #d8d8d4;
    }
    th, td {
      padding: 10px 12px;
      border-bottom: 1px solid #e5e5e1;
      text-align: left;
      vertical-align: top;
      font-size: 14px;
      line-height: 1.4;
      overflow-wrap: anywhere;
    }
    th {
      width: 220px;
      color: #4b4b47;
      background: #fbfbf9;
      font-weight: 600;
    }
    tr:last-child th, tr:last-child td {
      border-bottom: 0;
    }
  </style>
</head>
<body>
  <main>
    <h1>WorkMuch Status</h1>
    <table>
` + rowHTML.String() + `    </table>
  </main>
</body>
</html>`
}

type htmlRow struct {
	label string
	value string
}

func writeLine(b *strings.Builder, label string, value string) {
	b.WriteString(label)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteByte('\n')
}

func valueOrUnavailable(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unavailable"
	}
	return value
}

func stringOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func windowTitleText(sample SampleReport) string {
	if sample.WindowTitleAvailable || sample.WindowTitle != "" {
		return sample.WindowTitle
	}
	return "unavailable"
}

func logDirectoryText(logs LogDirectoryReport) string {
	if logs.Directory == "" {
		return "unavailable"
	}
	if logs.Writable {
		return logs.Directory + " (writable)"
	}
	return logs.Directory + " (not writable)"
}

func launchAgentText(report LaunchAgentReport) string {
	value := stringOrUnknown(string(report.State))
	if report.Detail != "" {
		value += " - " + report.Detail
	}
	if report.Error != "" {
		value += " - " + report.Error
	}
	return value
}

func runtimeStatusText(report RuntimeStatusReport) string {
	if report.Error != "" {
		return "error"
	}
	if !report.Present {
		return "unknown"
	}
	if report.Status.StartedAt != nil && report.Status.StoppedAt == nil {
		return "running"
	}
	return "stopped"
}

func timeText(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "unavailable"
	}
	return value.Format(time.RFC3339)
}

func eventText(at time.Time, message string) string {
	if at.IsZero() {
		return message
	}
	return at.Format(time.RFC3339) + " - " + message
}
