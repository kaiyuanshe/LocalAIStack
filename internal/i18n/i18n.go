package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"gopkg.in/yaml.v3"
)

var (
	defaultService *Service
	defaultMu      sync.RWMutex
)

type Service struct {
	language     string
	localesDir   string
	translator   Translator
	translations map[string]map[string]string
	mu           sync.Mutex
}

func Init(cfg config.I18nConfig) error {
	service, err := NewService(cfg)
	if err != nil {
		return err
	}
	defaultMu.Lock()
	defaultService = service
	defaultMu.Unlock()
	return nil
}

func NewService(cfg config.I18nConfig) (*Service, error) {
	language := strings.TrimSpace(cfg.Language)
	if language == "" {
		language = "en"
	}
	service := &Service{
		language:     strings.ToLower(language),
		localesDir:   defaultLocalesDir(),
		translations: make(map[string]map[string]string),
	}
	if service.language != "en" {
		service.translator = NewLLMTranslator(cfg.Translation)
	}
	return service, nil
}

func T(key string, args ...any) string {
	defaultMu.RLock()
	service := defaultService
	defaultMu.RUnlock()
	if service == nil {
		return fmt.Sprintf(key, args...)
	}
	return service.T(key, args...)
}

func Errorf(key string, args ...any) error {
	return fmt.Errorf(T(key, args...))
}

func (s *Service) T(key string, args ...any) string {
	if s == nil {
		return fmt.Sprintf(key, args...)
	}
	if s.language == "" || s.language == "en" {
		return fmt.Sprintf(key, args...)
	}
	value := s.lookupTranslation(key)
	if value == "" {
		value = s.translateAndStore(key)
	}
	if value == "" {
		value = key
	}
	return fmt.Sprintf(value, args...)
}

func (s *Service) lookupTranslation(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	cache := s.ensureLocaleLoadedLocked()
	return strings.TrimSpace(cache[key])
}

func (s *Service) translateAndStore(key string) string {
	if s.translator == nil {
		return ""
	}
	translated, err := s.translator.Translate(key, "en", s.language)
	if err != nil {
		return ""
	}
	translated = strings.TrimSpace(translated)
	if translated == "" {
		return ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	cache := s.ensureLocaleLoadedLocked()
	cache[key] = translated
	if err := s.writeLocaleLocked(cache); err != nil {
		return translated
	}
	return translated
}

func (s *Service) ensureLocaleLoadedLocked() map[string]string {
	cache, ok := s.translations[s.language]
	if ok {
		return cache
	}
	cache = make(map[string]string)
	path := s.localePath()
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		_ = yaml.Unmarshal(data, &cache)
	}
	s.translations[s.language] = cache
	return cache
}

func (s *Service) writeLocaleLocked(cache map[string]string) error {
	if s.localesDir == "" {
		return nil
	}
	if err := os.MkdirAll(s.localesDir, 0o755); err != nil {
		return err
	}
	payload, err := yaml.Marshal(cache)
	if err != nil {
		return err
	}
	return os.WriteFile(s.localePath(), payload, 0o644)
}

func (s *Service) localePath() string {
	return filepath.Join(s.localesDir, s.language+".yaml")
}

func defaultLocalesDir() string {
	if env := strings.TrimSpace(os.Getenv("LOCALAISTACK_LOCALES_DIR")); env != "" {
		return env
	}
	if cwd, err := os.Getwd(); err == nil {
		path := filepath.Join(cwd, "locales")
		if dirExists(path) {
			return path
		}
	}
	if exe, err := os.Executable(); err == nil {
		path := filepath.Join(filepath.Dir(exe), "locales")
		if dirExists(path) {
			return path
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, "locales")
	}
	return "locales"
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
