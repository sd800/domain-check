package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sd800/domain-check/internal/output"
	"github.com/sd800/domain-check/internal/safeio"
)

const automaticLanguage output.Language = "auto"

func languageConfirmation(preference, lang output.Language) string {
	switch preference {
	case output.English:
		return "Language switched to English and saved as the default."
	case output.Chinese:
		return "语言已切换为简体中文，并保存为默认语言。"
	default:
		return output.Text(lang, "Language set to follow the device language. Preference saved.", "语言已切换为跟随设备语言，并保存设置。")
	}
}

type preferences struct {
	Language output.Language `json:"language"`
}

func preferencesPath() (string, error) {
	if dir := os.Getenv("DOMAIN_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "config.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "domain-check", "config.json"), nil
}

func loadLanguagePreference() (output.Language, error) {
	path, err := preferencesPath()
	if err != nil {
		return "", err
	}
	data, _, err := safeio.ReadFile(path, 4096)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var p preferences
	if err := json.Unmarshal(data, &p); err != nil {
		return "", err
	}
	switch p.Language {
	case output.English, output.Chinese, automaticLanguage:
		return p.Language, nil
	default:
		return "", fmt.Errorf("invalid saved language")
	}
}

func saveLanguagePreference(language output.Language) error {
	path, err := preferencesPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(preferences{Language: language}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".domain-preferences-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
