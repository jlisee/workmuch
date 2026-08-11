package doctor

import (
	"fmt"
	"html"
	"strings"
	"time"
)

const recentSaveThreshold = 5 * time.Second

func RenderText(report DoctorReport) string {
	var b strings.Builder
	b.WriteString("WorkMuch Doctor\n")
	writeHealthText(&b, report)
	writeLine(&b, "Selected backend", valueOrUnavailable(report.SelectedBackend))
	writeLine(&b, "Active backend", valueOrUnavailable(report.ActiveBackend))
	writeLinuxDiagnostics(&b, report)
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
	writeLine(&b, "Sample source", stringOrUnknown(string(report.Sample.Source)))
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
	writeLine(&b, "Login Item", loginItemText(report.LoginItem))
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
	}
	rows = appendLinuxDiagnosticRows(rows, report)
	rows = append(rows,
		htmlRow{"Accessibility permission", stringOrUnknown(string(report.Permission.State))},
		htmlRow{"Sample source", stringOrUnknown(string(report.Sample.Source))},
		htmlRow{"Frontmost app", valueOrUnavailable(report.Sample.FrontmostApp)},
		htmlRow{"Focused window title", windowTitleText(report.Sample)},
		htmlRow{"Idle seconds", fmt.Sprintf("%.2f", report.Sample.IdleSeconds)},
		htmlRow{"Log directory", logDirectoryText(report.Logs)},
		htmlRow{"Current work log", valueOrUnavailable(report.Logs.CurrentWorkLog)},
		htmlRow{"Login Item", loginItemText(report.LoginItem)},
		htmlRow{"Runtime status", runtimeStatusText(report.Runtime)},
		htmlRow{"Runtime status file", valueOrUnavailable(report.Runtime.Path)},
		htmlRow{"Runtime selected backend", valueOrUnavailable(report.Runtime.Status.SelectedBackend)},
		htmlRow{"Runtime active backend", valueOrUnavailable(report.Runtime.Status.ActiveBackend)},
		htmlRow{"Sample count", fmt.Sprintf("%d", report.Runtime.Status.SampleCount)},
		htmlRow{"Last started", timeText(report.Runtime.Status.StartedAt)},
		htmlRow{"Last stopped", timeText(report.Runtime.Status.StoppedAt)},
		htmlRow{"Last sample", timeText(report.Runtime.Status.LastSampleAt)},
		htmlRow{"Last successful sample", timeText(report.Runtime.Status.LastSuccessfulSampleAt)},
		htmlRow{"Runtime work log", valueOrUnavailable(report.Runtime.Status.CurrentWorkLogPath)},
	)
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

	var healthHTML strings.Builder
	for _, check := range healthChecks(report) {
		className := "health-fail"
		statusText := "FAIL"
		statusLabel := "FAIL"
		if check.ok {
			className = "health-ok"
			statusText = "OK"
			statusLabel = "OK"
		}
		healthHTML.WriteString("<tr class=\"")
		healthHTML.WriteString(className)
		healthHTML.WriteString("\"><th>")
		healthHTML.WriteString(html.EscapeString(check.label))
		healthHTML.WriteString("</th><td><span class=\"health-icon\" aria-label=\"")
		healthHTML.WriteString(statusLabel)
		healthHTML.WriteString("\">●</span> ")
		healthHTML.WriteString(statusText)
		healthHTML.WriteString("</td><td>")
		healthHTML.WriteString(html.EscapeString(check.detail))
		healthHTML.WriteString("</td></tr>\n")
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
    h2 {
      margin: 24px 0 10px;
      font-size: 18px;
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
    .health {
      margin-bottom: 22px;
    }
    .health th {
      width: 180px;
    }
    .health-icon {
      font-size: 18px;
      line-height: 1;
      vertical-align: -1px;
    }
    .health-ok .health-icon {
      color: #16833a;
    }
    .health-fail .health-icon {
      color: #c92a2a;
    }
  </style>
</head>
<body>
  <main>
    <h1>WorkMuch Status</h1>
    <h2>Health</h2>
    <table class="health">
` + healthHTML.String() + `    </table>
    <h2>Details</h2>
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

type healthCheck struct {
	label  string
	ok     bool
	detail string
}

func writeHealthText(b *strings.Builder, report DoctorReport) {
	for _, check := range healthChecks(report) {
		icon := "❌"
		if check.ok {
			icon = "✅"
		}
		b.WriteString(icon)
		b.WriteByte(' ')
		b.WriteString(check.label)
		b.WriteString(": ")
		b.WriteString(check.detail)
		b.WriteByte('\n')
	}
}

func healthChecks(report DoctorReport) []healthCheck {
	lastSaveOK, lastSaveDetail := lastSaveHealth(report)
	savingOK, savingDetail := savingHealth(report, lastSaveOK, lastSaveDetail)
	return []healthCheck{
		runningHealth(report),
		{label: "Last save <5s", ok: lastSaveOK, detail: lastSaveDetail},
		fieldHealth("Title works", report.Sample.WindowTitle),
		fieldHealth("App works", report.Sample.FrontmostApp),
		{label: "Saving works", ok: savingOK, detail: savingDetail},
	}
}

func runningHealth(report DoctorReport) healthCheck {
	check := healthCheck{label: "Running"}
	switch {
	case report.Runtime.Error != "":
		check.detail = "error"
	case !report.Runtime.Present:
		check.detail = "unknown"
	case report.Runtime.Status.StartedAt != nil && report.Runtime.Status.StoppedAt == nil:
		check.ok = true
		check.detail = "ok"
	default:
		check.detail = "stopped"
	}
	return check
}

func lastSaveHealth(report DoctorReport) (bool, string) {
	if report.Runtime.Status.LastSuccessfulSampleAt == nil ||
		report.Runtime.Status.LastSuccessfulSampleAt.IsZero() {
		return false, "unavailable"
	}
	age := reportTime(report).Sub(*report.Runtime.Status.LastSuccessfulSampleAt)
	if age <= recentSaveThreshold {
		return true, "ok"
	}
	return false, "stale"
}

func fieldHealth(label string, value string) healthCheck {
	if strings.TrimSpace(value) == "" {
		return healthCheck{label: label, detail: "unavailable"}
	}
	return healthCheck{label: label, ok: true, detail: "ok"}
}

func savingHealth(report DoctorReport, lastSaveOK bool, lastSaveDetail string) (bool, string) {
	if strings.TrimSpace(report.Runtime.Status.CurrentWorkLogPath) == "" {
		return false, "unavailable"
	}
	if !lastSaveOK {
		return false, lastSaveDetail
	}
	return true, "ok"
}

func reportTime(report DoctorReport) time.Time {
	if report.GeneratedAt.IsZero() {
		return time.Now()
	}
	return report.GeneratedAt
}

func writeLinuxDiagnostics(b *strings.Builder, report DoctorReport) {
	if report.LinuxSession.Applicable {
		writeLine(b, "Desktop session", valueOrUnavailable(report.LinuxSession.Type))
		writeLine(b, "Linux session support", stringOrUnknown(string(report.LinuxSession.Support)))
		if report.LinuxSession.Detail != "" {
			writeLine(b, "Linux session detail", report.LinuxSession.Detail)
		}
		if report.LinuxSession.WaylandDisplay != "" {
			writeLine(b, "Wayland display", report.LinuxSession.WaylandDisplay)
		}
	}
	if report.X11.Applicable {
		writeLine(b, "X11 display", valueOrUnavailable(report.X11.Display))
		writeLine(b, "X11 connection", stringOrUnknown(string(report.X11.Connection)))
		if report.X11.ConnectionError != "" {
			writeLine(b, "X11 connection error", report.X11.ConnectionError)
		}
		writeLine(b, "X11 sampling", stringOrUnknown(string(report.X11.Sampling)))
		if report.X11.SamplingError != "" {
			writeLine(b, "X11 sampling error", report.X11.SamplingError)
		}
	}
}

func appendLinuxDiagnosticRows(rows []htmlRow, report DoctorReport) []htmlRow {
	if report.LinuxSession.Applicable {
		rows = append(rows,
			htmlRow{"Desktop session", valueOrUnavailable(report.LinuxSession.Type)},
			htmlRow{"Linux session support", stringOrUnknown(string(report.LinuxSession.Support))},
		)
		if report.LinuxSession.Detail != "" {
			rows = append(rows, htmlRow{"Linux session detail", report.LinuxSession.Detail})
		}
		if report.LinuxSession.WaylandDisplay != "" {
			rows = append(rows, htmlRow{"Wayland display", report.LinuxSession.WaylandDisplay})
		}
	}
	if report.X11.Applicable {
		rows = append(rows,
			htmlRow{"X11 display", valueOrUnavailable(report.X11.Display)},
			htmlRow{"X11 connection", stringOrUnknown(string(report.X11.Connection))},
		)
		if report.X11.ConnectionError != "" {
			rows = append(rows, htmlRow{"X11 connection error", report.X11.ConnectionError})
		}
		rows = append(rows, htmlRow{"X11 sampling", stringOrUnknown(string(report.X11.Sampling))})
		if report.X11.SamplingError != "" {
			rows = append(rows, htmlRow{"X11 sampling error", report.X11.SamplingError})
		}
	}
	return rows
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

func loginItemText(report LoginItemReport) string {
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
