package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/system"
	"gopkg.in/yaml.v3"
)

func newInitCommand() *cobra.Command {
	var apiKey string
	var language string
	var assistantProvider string
	var assistantModel string
	var assistantBaseURL string
	var assistantTimeoutSeconds int
	var translationProvider string
	var translationModel string
	var translationBaseURL string
	var translationTimeoutSeconds int
	var configPath string

	initCmd := &cobra.Command{
		Use:   "init",
		Short: i18n.T("Initialize LocalAIStack interactive configuration"),
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
			existingAPIKey := readNestedString(settings, "llm", "api_key")
			if existingAPIKey == "" {
				existingAPIKey = readNestedString(settings, "i18n", "translation", "api_key")
			}
			existingLanguage := readNestedString(settings, "i18n", "language")

			if apiKey == "" {
				apiKey, err = promptValueWithDefault(reader, i18n.T("SiliconFlow API Key"), existingAPIKey, maskAPIKey(existingAPIKey))
				if err != nil {
					return err
				}
			}

			if language == "" {
				language, err = promptValue(reader, i18n.T("Preferred language"), fallbackValue(existingLanguage, "en"))
				if err != nil {
					return err
				}
			}

			if language == "" {
				return i18n.Errorf("language cannot be empty")
			}

			setNestedValue(settings, apiKey, "llm", "api_key")
			setNestedValue(settings, assistantProvider, "llm", "provider")
			setNestedValue(settings, assistantModel, "llm", "model")
			setNestedValue(settings, assistantBaseURL, "llm", "base_url")
			setNestedValue(settings, assistantTimeoutSeconds, "llm", "timeout_seconds")
			setNestedValue(settings, language, "i18n", "language")
			setNestedValue(settings, translationProvider, "i18n", "translation", "provider")
			setNestedValue(settings, translationModel, "i18n", "translation", "model")
			setNestedValue(settings, apiKey, "i18n", "translation", "api_key")
			setNestedValue(settings, translationBaseURL, "i18n", "translation", "base_url")
			setNestedValue(settings, translationTimeoutSeconds, "i18n", "translation", "timeout_seconds")

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

			cmd.Printf("%s\n", i18n.T("Configuration written to %s", configPath))

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			baseInfoPath := filepath.Join(homeDir, ".localaistack", "base_info.json")
			if err := system.WriteBaseInfo("", "json", true, false); err != nil {
				return err
			}
			cmd.Printf("%s\n", i18n.T("Base system info written to %s", baseInfoPath))
			return nil
		},
	}

	initCmd.Flags().StringVar(&configPath, "config-path", "", i18n.T("config file path (default is ~/.localaistack/config.yaml)"))
	initCmd.Flags().StringVar(&apiKey, "api-key", "", i18n.T("SiliconFlow API key"))
	initCmd.Flags().StringVar(&language, "language", "", i18n.T("Preferred interaction language"))
	initCmd.Flags().StringVar(&assistantProvider, "assistant-provider", "siliconflow", i18n.T("Assistant provider"))
	initCmd.Flags().StringVar(&assistantModel, "assistant-model", "deepseek-ai/DeepSeek-V3.2", i18n.T("Assistant model"))
	initCmd.Flags().StringVar(&assistantBaseURL, "assistant-base-url", "https://api.siliconflow.cn/v1/chat/completions", i18n.T("Assistant API base URL"))
	initCmd.Flags().IntVar(&assistantTimeoutSeconds, "assistant-timeout-seconds", 30, i18n.T("Assistant timeout in seconds"))
	initCmd.Flags().StringVar(&translationProvider, "translation-provider", "siliconflow", i18n.T("Translation provider"))
	initCmd.Flags().StringVar(&translationModel, "translation-model", "tencent/Hunyuan-MT-7B", i18n.T("Translation model"))
	initCmd.Flags().StringVar(&translationBaseURL, "translation-base-url", "https://api.siliconflow.cn/v1/chat/completions", i18n.T("Translation API base URL"))
	initCmd.Flags().IntVar(&translationTimeoutSeconds, "translation-timeout-seconds", 30, i18n.T("Translation timeout in seconds"))
	initCmd.Flags().StringVar(&translationProvider, "provider", "siliconflow", i18n.T("Translation provider"))
	initCmd.Flags().StringVar(&translationModel, "model", "tencent/Hunyuan-MT-7B", i18n.T("Translation model"))
	initCmd.Flags().StringVar(&translationBaseURL, "base-url", "https://api.siliconflow.cn/v1/chat/completions", i18n.T("Translation API base URL"))
	initCmd.Flags().IntVar(&translationTimeoutSeconds, "timeout-seconds", 30, i18n.T("Translation timeout in seconds"))
	_ = initCmd.Flags().MarkDeprecated("provider", "use --translation-provider")
	_ = initCmd.Flags().MarkDeprecated("model", "use --translation-model")
	_ = initCmd.Flags().MarkDeprecated("base-url", "use --translation-base-url")
	_ = initCmd.Flags().MarkDeprecated("timeout-seconds", "use --translation-timeout-seconds")

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
	return promptValueWithDefault(reader, label, defaultValue, "")
}

func promptValueWithDefault(reader *bufio.Reader, label string, defaultValue string, displayValue string) (string, error) {
	promptValue := defaultValue
	if displayValue != "" {
		promptValue = displayValue
	}
	if promptValue != "" {
		fmt.Printf("%s", i18n.T("%s [%s]: ", label, promptValue))
	} else {
		fmt.Printf("%s", i18n.T("%s: ", label))
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

func maskAPIKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	const suffixLength = 4
	prefix := ""
	if strings.HasPrefix(trimmed, "sk-") {
		prefix = "sk-"
	}
	if len(trimmed) <= len(prefix)+suffixLength {
		return strings.Repeat("*", len(trimmed))
	}
	return prefix + strings.Repeat("*", len(trimmed)-len(prefix)-suffixLength) + trimmed[len(trimmed)-suffixLength:]
}
