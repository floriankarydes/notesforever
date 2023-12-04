package service

import (
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
)

type plistParams struct {
	Name string
	Prog string
	Args []string
	Hour int
}

const plistHeader = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
`
const plistTemplateSrc = `
<plist version="1.0">
  <dict>

  	<key>Label</key>
    <string>com.{{ .Name }}.agent</string>

		<key>ProgramArguments</key>
    <array>
      <string>{{ .Prog }}</string>
			{{range .Args}}<string>{{ . }}</string>{{end}}
    </array>

		<key>StandardOutPath</key>
		<string>/tmp/{{ .Name }}.stdout</string>
		<key>StandardErrorPath</key>
		<string>/tmp/{{ .Name }}.stderr</string>

		<key>WorkingDirectory</key>
		<string>/tmp</string>

    <key>RunAtLoad</key>
    <false/>

    <key>StartCalendarInterval</key>
		<array>
			<dict>
				<key>Hour</key>
				<integer>{{ .Hour }}</integer>
				<key>Minute</key>
				<integer>0</integer>
			</dict>
		</array>

  </dict>
</plist>
`

var plistTemplate = template.Must(template.New("plistTemplate").Parse(plistTemplateSrc))

// Install a launchd service to run every day.
func RunEverydayAt(hour int, name string, args ...string) error {

	// Prepare plist params.
	prog, err := exec.LookPath(name)
	if err != nil {
		return err
	}
	params := plistParams{
		Name: name,
		Prog: prog,
		Args: args,
		Hour: hour,
	}

	// Create plist file.
	userPath, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	fileName := "com." + name + "agent.plist"
	filePath := filepath.Join(userPath, "Library", "LaunchAgents", fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(plistHeader); err != nil {
		return err
	}
	if err := plistTemplate.Execute(file, params); err != nil {
		return err
	}

	// Start service.
	if err := exec.Command("launchctl", "load", filePath).Run(); err != nil {
		return err
	}

	return nil
}
