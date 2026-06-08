package express

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/recipe"
)

func Preflight(record recipe.Record) PreflightReport {
	r := record.Recipe
	report := PreflightReport{RecipeID: r.ID}

	report.Checks = append(report.Checks, checkRecipeArtifacts(record)...)
	report.Checks = append(report.Checks, localModelCheck(record)...)
	if r.Kind != recipe.KindInference {
		report.Checks = append(report.Checks, CheckResult{
			Name:       "recipe.kind",
			Status:     CheckFail,
			Message:    "Express Inference commands require an inference recipe.",
			Suggestion: "Use an inference recipe under recipes/express/inference.",
		})
		return report
	}

	if r.Target.Runtime.Docker {
		report.Checks = append(report.Checks, commandCheck("docker", "Docker CLI is available.", "Install Docker before running Docker recipes."))
		report.Checks = append(report.Checks, dockerComposeCheck())
	}
	if r.Target.Runtime.Native {
		report.Checks = append(report.Checks, CheckResult{
			Name:    "native-runtime",
			Status:  CheckPass,
			Message: "Recipe allows native runtime.",
		})
	}
	if strings.EqualFold(r.Target.Hardware.Vendor, "nvidia") {
		report.Checks = append(report.Checks, commandCheck("nvidia-smi", "NVIDIA tooling is available.", "Install NVIDIA drivers and NVIDIA Container Toolkit."))
		report.Checks = append(report.Checks, dockerGPUCheck())
	}
	if r.Target.Runtime.Metal && runtime.GOOS != "darwin" {
		report.Checks = append(report.Checks, CheckResult{
			Name:       "metal",
			Status:     CheckWarn,
			Message:    "Metal runtime is only available on macOS.",
			Suggestion: "Run this recipe on Apple Silicon macOS or choose another recipe.",
		})
	}
	if r.Runtime.Port > 0 {
		report.Checks = append(report.Checks, portCheck(r.Runtime.Port))
	}

	return report
}

func localModelCheck(record recipe.Record) []CheckResult {
	r := record.Recipe
	if !strings.EqualFold(r.Model.Source, "local") && !strings.EqualFold(r.Model.Format, "gguf") {
		return nil
	}
	modelPath := resolveLocalModelPath(record)
	if modelPath == "" {
		return nil
	}
	if info, err := os.Stat(modelPath); err == nil {
		if info.IsDir() {
			return []CheckResult{{
				Name:       "model.local_file",
				Status:     CheckFail,
				Message:    fmt.Sprintf("Local model path is a directory: %s", modelPath),
				Suggestion: "Set MODEL_FILE to a GGUF filename or update recipe model.id to a model file.",
			}}
		}
		return []CheckResult{{
			Name:    "model.local_file",
			Status:  CheckPass,
			Message: fmt.Sprintf("Local model file exists: %s", modelPath),
		}}
	} else if !os.IsNotExist(err) {
		return []CheckResult{{
			Name:       "model.local_file",
			Status:     CheckFail,
			Message:    fmt.Sprintf("Cannot access local model file %s: %v", modelPath, err),
			Suggestion: "Fix file permissions or choose a readable model path.",
		}}
	}
	return []CheckResult{{
		Name:       "model.local_file",
		Status:     CheckFail,
		Message:    fmt.Sprintf("Local model file is missing: %s", modelPath),
		Suggestion: fmt.Sprintf("Place a GGUF model at %s, or set MODEL_DIR/MODEL_FILE before running this recipe.", modelPath),
	}}
}

func resolveLocalModelPath(record recipe.Record) string {
	dir := filepath.Dir(record.SourcePath)
	modelDir := strings.TrimSpace(os.Getenv("MODEL_DIR"))
	modelFile := strings.TrimSpace(os.Getenv("MODEL_FILE"))
	if modelFile != "" {
		if filepath.IsAbs(modelFile) {
			return modelFile
		}
		if modelDir == "" {
			modelDir = filepath.Join(dir, "models")
		}
		return filepath.Join(modelDir, modelFile)
	}
	id := strings.TrimSpace(record.Recipe.Model.ID)
	if id == "" {
		return ""
	}
	if strings.HasPrefix(id, "~/") {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			id = filepath.Join(home, strings.TrimPrefix(id, "~/"))
		}
	}
	if filepath.IsAbs(id) {
		return id
	}
	if modelDir != "" {
		return filepath.Join(modelDir, filepath.Base(id))
	}
	return filepath.Join(dir, id)
}

func checkRecipeArtifacts(record recipe.Record) []CheckResult {
	dir := filepath.Dir(record.SourcePath)
	required := []string{"recipe.yaml"}
	optional := []string{"docker-compose.yaml", ".env.example", "healthcheck.sh", "benchmark.sh"}
	var checks []CheckResult
	for _, name := range required {
		checks = append(checks, fileCheck(filepath.Join(dir, name), name, true))
	}
	for _, name := range optional {
		checks = append(checks, fileCheck(filepath.Join(dir, name), name, false))
	}
	return checks
}

func fileCheck(path, name string, required bool) CheckResult {
	if _, err := os.Stat(path); err == nil {
		return CheckResult{Name: "artifact." + name, Status: CheckPass, Message: name + " exists."}
	} else if !os.IsNotExist(err) {
		return CheckResult{Name: "artifact." + name, Status: CheckFail, Message: err.Error()}
	}
	if required {
		return CheckResult{Name: "artifact." + name, Status: CheckFail, Message: name + " is missing."}
	}
	return CheckResult{Name: "artifact." + name, Status: CheckWarn, Message: name + " is not present yet."}
}

func commandCheck(name, okMessage, suggestion string) CheckResult {
	if _, err := exec.LookPath(name); err == nil {
		return CheckResult{Name: "command." + name, Status: CheckPass, Message: okMessage}
	}
	return CheckResult{Name: "command." + name, Status: CheckFail, Message: name + " was not found in PATH.", Suggestion: suggestion}
}

func dockerComposeCheck() CheckResult {
	if _, err := exec.LookPath("docker"); err != nil {
		return CheckResult{Name: "docker.compose", Status: CheckFail, Message: "Docker CLI was not found.", Suggestion: "Install Docker with Compose support."}
	}
	if err := exec.Command("docker", "compose", "version").Run(); err == nil {
		return CheckResult{Name: "docker.compose", Status: CheckPass, Message: "Docker Compose plugin is available."}
	}
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return CheckResult{Name: "docker.compose", Status: CheckPass, Message: "docker-compose is available."}
	}
	return CheckResult{Name: "docker.compose", Status: CheckFail, Message: "Docker Compose was not found.", Suggestion: "Install Docker Compose plugin or docker-compose."}
}

func dockerGPUCheck() CheckResult {
	if _, err := exec.LookPath("docker"); err != nil {
		return CheckResult{Name: "docker.gpu", Status: CheckFail, Message: "Docker CLI was not found.", Suggestion: "Install Docker and NVIDIA Container Toolkit."}
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		return CheckResult{Name: "docker.gpu", Status: CheckWarn, Message: "Docker daemon is not reachable.", Suggestion: "Start Docker and retry preflight."}
	}
	if _, err := exec.LookPath("nvidia-container-cli"); err == nil {
		return CheckResult{Name: "docker.gpu", Status: CheckPass, Message: "NVIDIA Container Toolkit CLI is available."}
	}
	return CheckResult{
		Name:       "docker.gpu",
		Status:     CheckWarn,
		Message:    "NVIDIA Container Toolkit CLI was not found.",
		Suggestion: "Install/configure NVIDIA Container Toolkit, then run nvidia-ctk runtime configure --runtime=docker. Use a real GPU smoke test before marking the recipe verified.",
	}
}

func portCheck(port int) CheckResult {
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "operation not permitted") {
			return CheckResult{
				Name:       "port." + strconv.Itoa(port),
				Status:     CheckWarn,
				Message:    fmt.Sprintf("Port %d could not be probed in this environment: %v", port, err),
				Suggestion: "Run preflight outside a restricted sandbox to confirm port availability.",
			}
		}
		return CheckResult{
			Name:       "port." + strconv.Itoa(port),
			Status:     CheckFail,
			Message:    fmt.Sprintf("Port %d is not available: %v", port, err),
			Suggestion: "Stop the process using the port or override the recipe port.",
		}
	}
	_ = listener.Close()
	return CheckResult{Name: "port." + strconv.Itoa(port), Status: CheckPass, Message: fmt.Sprintf("Port %d is available.", port)}
}
