package localweb

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sd800/domain-check/internal/registry"
	"github.com/sd800/domain-check/internal/safeio"
)

const serviceLabel = "localhost.domain-check.web"

type Settings struct {
	Enabled bool `json:"enabled"`
	AtLogin bool `json:"at_login"`
}

type Status struct {
	Settings
	Running bool
}

// Manager keeps setup choices and launchd registration separate from query data.
type Manager struct {
	ConfigDir  string
	Home       string
	Executable string
	Version    string
	CacheDir   string
	Platform   string
	Command    func(context.Context, ...string) error
	Probe      func(context.Context) (string, bool)
	Available  func() error
}

func NewManager(version string) (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	config := os.Getenv("DOMAIN_CONFIG_DIR")
	if config == "" {
		config, err = os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		config = filepath.Join(config, "domain-check")
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	config, err = filepath.Abs(config)
	if err != nil {
		return nil, err
	}
	cache, err := filepath.Abs(registry.DefaultCacheDir())
	if err != nil {
		return nil, err
	}
	m := &Manager{ConfigDir: config, Home: home, Executable: executable, Version: version, CacheDir: cache, Platform: runtime.GOOS}
	m.Command = func(ctx context.Context, args ...string) error {
		ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "/bin/launchctl", args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("launchctl %s failed: %w", args[0], err)
		}
		return nil
	}
	m.Probe = probeService
	m.Available = func() error {
		for _, address := range []string{"127.0.0.1:20771", "[::1]:20771"} {
			l, err := net.Listen("tcp", address)
			if err != nil {
				return fmt.Errorf("local port 20771 is already in use")
			}
			_ = l.Close()
		}
		return nil
	}
	return m, nil
}

func (m *Manager) settingsPath() string { return filepath.Join(m.ConfigDir, "web.json") }
func (m *Manager) agentPath(login bool) string {
	if login {
		return filepath.Join(m.Home, "Library", "LaunchAgents", serviceLabel+".plist")
	}
	return filepath.Join(m.ConfigDir, "web-agent.plist")
}
func (m *Manager) target() string { return "gui/" + strconv.Itoa(os.Getuid()) }
func (m *Manager) job() string    { return m.target() + "/" + serviceLabel }

func (m *Manager) Load() (Settings, error) {
	data, _, err := safeio.ReadFile(m.settingsPath(), 4096)
	if os.IsNotExist(err) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	var settings Settings
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&settings); err != nil {
		return Settings{}, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return Settings{}, fmt.Errorf("invalid web settings")
	}
	return settings, nil
}

func (m *Manager) Status(ctx context.Context) (Status, error) {
	settings, err := m.Load()
	if err != nil {
		return Status{}, err
	}
	_, running := m.Probe(ctx)
	return Status{Settings: settings, Running: running}, nil
}

func atomicFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".domain-web-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func (m *Manager) save(settings Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return atomicFile(m.settingsPath(), append(data, '\n'))
}

func escapeXML(value string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}

func (m *Manager) plist() []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>` + serviceLabel + `</string>
<key>ProgramArguments</key><array><string>` + escapeXML(m.Executable) + `</string><string>--web-serve</string></array>
<key>RunAtLoad</key><true/>
<key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
<key>ThrottleInterval</key><integer>10</integer>
<key>ProcessType</key><string>Background</string>
<key>EnvironmentVariables</key><dict>
<key>DOMAIN_CONFIG_DIR</key><string>` + escapeXML(m.ConfigDir) + `</string>
<key>DOMAIN_CACHE_DIR</key><string>` + escapeXML(m.CacheDir) + `</string>
</dict>
<key>StandardOutPath</key><string>/dev/null</string>
<key>StandardErrorPath</key><string>/dev/null</string>
</dict></plist>
`)
}

func (m *Manager) stop(ctx context.Context) error {
	if m.Command(ctx, "print", m.job()) != nil {
		return nil
	}
	return m.Command(ctx, "bootout", m.job())
}

func (m *Manager) agentMatches(settings Settings) (bool, error) {
	data, _, err := safeio.ReadFile(m.agentPath(settings.AtLogin), 32768)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return bytes.Equal(data, m.plist()), nil
}

func (m *Manager) start(ctx context.Context, settings Settings) error {
	matches, err := m.agentMatches(settings)
	if err != nil {
		return err
	}
	if !matches {
		if err := atomicFile(m.agentPath(settings.AtLogin), m.plist()); err != nil {
			return err
		}
	}
	if err := m.Command(ctx, "enable", m.job()); err != nil {
		return err
	}
	if err := m.Command(ctx, "bootstrap", m.target(), m.agentPath(settings.AtLogin)); err != nil {
		return err
	}
	return m.waitReady(ctx)
}

func (m *Manager) waitReady(ctx context.Context) error {
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		if version, running := m.Probe(ctx); running && version == m.Version {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("the local web service did not start; check port 20771 and the installed executable")
		case <-tick.C:
		}
	}
}

// Apply changes only this user's job and stores the choices after a successful transition.
func (m *Manager) Apply(ctx context.Context, next Settings) error {
	if m.Platform != "darwin" {
		return fmt.Errorf("web setup requires macOS")
	}
	previous, err := m.Load()
	if err != nil {
		return err
	}
	if !next.Enabled {
		next.AtLogin = false
	}
	if next == previous && next.Enabled {
		matches, err := m.agentMatches(next)
		if err != nil {
			return err
		}
		if matches {
			if _, running := m.Probe(ctx); running {
				return nil
			}
			if m.Command(ctx, "print", m.job()) == nil {
				if err := m.Command(ctx, "kickstart", "-k", m.job()); err != nil {
					return err
				}
				return m.waitReady(ctx)
			}
			if err := m.Available(); err != nil {
				return err
			}
			return m.start(ctx, next)
		}
	}
	if !next.Enabled {
		if err = m.stop(ctx); err != nil {
			return err
		}
		if next != previous {
			if err = m.save(next); err != nil {
				return err
			}
		}
		for _, login := range []bool{false, true} {
			if err = os.Remove(m.agentPath(login)); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		return nil
	}
	if err = m.stop(ctx); err != nil {
		return err
	}
	// launchd may still be closing sockets immediately after bootout.
	for attempt := 0; attempt < 20; attempt++ {
		err = m.Available()
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	if err != nil {
		if previous.Enabled {
			_ = m.start(ctx, previous)
		}
		return err
	}
	if err = m.start(ctx, next); err != nil {
		_ = m.stop(ctx)
		_ = os.Remove(m.agentPath(next.AtLogin))
		if previous.Enabled {
			_ = m.start(ctx, previous)
		}
		return err
	}
	if err = m.save(next); err != nil {
		_ = m.stop(ctx)
		_ = os.Remove(m.agentPath(next.AtLogin))
		if previous.Enabled {
			_ = m.start(ctx, previous)
		}
		return err
	}
	if previous.AtLogin != next.AtLogin {
		_ = os.Remove(m.agentPath(previous.AtLogin))
	}
	return nil
}

func (m *Manager) SetEnabled(ctx context.Context, on bool) error {
	settings, err := m.Load()
	if err != nil {
		return err
	}
	settings.Enabled = on
	return m.Apply(ctx, settings)
}
func (m *Manager) SetLogin(ctx context.Context, on bool) error {
	settings, err := m.Load()
	if err != nil {
		return err
	}
	settings.AtLogin = on
	if on {
		settings.Enabled = true
	}
	return m.Apply(ctx, settings)
}

// Refresh reuses an existing job after an executable update and never registers a login item.
func (m *Manager) Refresh(ctx context.Context) error {
	settings, err := m.Load()
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return nil
	}
	if version, running := m.Probe(ctx); running && version == m.Version {
		return nil
	}
	// An unloaded job may have been disabled in macOS settings. Only explicit setup
	// may register it again; installing or running the CLI must leave it alone.
	if m.Command(ctx, "print", m.job()) != nil {
		return nil
	}
	matches, err := m.agentMatches(settings)
	if err != nil {
		return err
	}
	if !matches {
		return fmt.Errorf("web service configuration has changed; run domain setup to apply it")
	}
	if err := m.Command(ctx, "kickstart", "-k", m.job()); err != nil {
		return err
	}
	return m.waitReady(ctx)
}

func probeService(ctx context.Context) (string, bool) {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	transport := &http.Transport{Proxy: nil}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 500 * time.Millisecond, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:20771/api/health", nil)
	if err != nil {
		return "", false
	}
	request.Host = Address
	response, err := client.Do(request)
	if err != nil {
		return "", false
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || response.Header.Get("X-Domain-Check") != "registry-local-interface" {
		return "", false
	}
	var health struct {
		Version string `json:"version"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&health) != nil {
		return "", false
	}
	return strings.TrimSpace(health.Version), true
}
