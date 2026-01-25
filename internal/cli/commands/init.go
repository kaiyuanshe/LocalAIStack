package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"gopkg.in/yaml.v3"
)

func newInitCommand() *cobra.Command {
	var apiKey string
	var language string
	var provider string
	var model string
	var baseURL string
	var timeoutSeconds int
	var configPath string

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize LocalAIStack interactive configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				defaultPath, err := config.DefaultUserConfigPath()
				if err != nil {
					return err
				}
				configPath = defaultPath
			}

			settings, err := loadConfigMap(configPath)
			if err != nil {
				return err
			}

			reader := bufio.NewReader(cmd.InOrStdin())
			existingAPIKey := readNestedString(settings, "i18n", "translation", "api_key")
			existingLanguage := readNestedString(settings, "i18n", "language")

			if apiKey == "" {
				apiKey, err = promptValue(reader, "SiliconFlow API Key", existingAPIKey)
				if err != nil {
					return err
				}
			}

			if language == "" {
				language, err = promptValue(reader, "Preferred language", fallbackValue(existingLanguage, "en"))
				if err != nil {
					return err
				}
			}

			if language == "" {
				return fmt.Errorf("language cannot be empty")
			}

			setNestedValue(settings, apiKey, "i18n", "translation", "api_key")
			setNestedValue(settings, language, "i18n", "language")
			setNestedValue(settings, provider, "i18n", "translation", "provider")
			setNestedValue(settings, model, "i18n", "translation", "model")
			setNestedValue(settings, baseURL, "i18n", "translation", "base_url")
			setNestedValue(settings, timeoutSeconds, "i18n", "translation", "timeout_seconds")

			if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
				return err
			}

			payload, err := yaml.Marshal(settings)
			if err != nil {
				return err
			}

			if err := os.WriteFile(configPath, payload, 0o600); err != nil {
				return err
			}

			cmd.Printf("Configuration written to %s\n", configPath)
			return nil
		},
	}

	initCmd.Flags().StringVar(&configPath, "config-path", "", "config file path (default is ~/.localaistack/config.yaml)")
	initCmd.Flags().StringVar(&apiKey, "api-key", "", "SiliconFlow API key")
	initCmd.Flags().StringVar(&language, "language", "", "Preferred interaction language")
	initCmd.Flags().StringVar(&provider, "provider", "siliconflow", "Translation provider")
	initCmd.Flags().StringVar(&model, "model", "tencent/Hunyuan-MT-7B", "Translation model")
	initCmd.Flags().StringVar(&baseURL, "base-url", "https://api.siliconflow.cn/v1/chat/completions", "Translation API base URL")
	initCmd.Flags().IntVar(&timeoutSeconds, "timeout-seconds", 30, "Translation timeout in seconds")

	return initCmd
}

func loadConfigMap(path string) (map[string]interface{}, error) {
	settings := map[string]interface{}{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return settings, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return settings, nil
	}

	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func readNestedString(data map[string]interface{}, path ...string) string {
	current := data
	for index, key := range path {
		value, ok := current[key]
		if !ok {
			return ""
		}
		if index == len(path)-1 {
			if stringValue, ok := value.(string); ok {
				return strings.TrimSpace(stringValue)
			}
			return ""
		}
		next, ok := value.(map[string]interface{})
		if !ok {
			return ""
		}
		current = next
	}
	return ""
}

func setNestedValue(data map[string]interface{}, value interface{}, path ...string) {
	current := data
	for index, key := range path {
		if index == len(path)-1 {
			current[key] = value
			return
		}
		next, ok := current[key].(map[string]interface{})
		if !ok {
			next = map[string]interface{}{}
			current[key] = next
		}
		current = next
	}
}

func promptValue(reader *bufio.Reader, label string, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}
	return input, nil
}

func fallbackValue(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
