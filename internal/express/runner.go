package express

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/recipe"
)

func BuildRunPlan(record recipe.Record) RunPlan {
	r := record.Recipe
	dir := filepath.Dir(record.SourcePath)
	composePath := filepath.Join(dir, "docker-compose.yaml")
	plan := RunPlan{
		RecipeID: r.ID,
		Mode:     r.Runtime.Mode,
		WorkDir:  dir,
	}
	if r.Target.Runtime.Docker && fileExists(composePath) {
		plan.Mode = "docker-compose"
		plan.Command = []string{"docker", "compose", "-f", composePath, "up", "-d"}
		return plan
	}
	if r.Target.Runtime.Docker {
		plan.Mode = "docker"
		if strings.TrimSpace(r.Runtime.Image) != "" {
			plan.Command = []string{"docker", "run", "--rm", "-p", portMapping(r.Runtime.Port), r.Runtime.Image}
		}
		plan.Notes = append(plan.Notes, "docker-compose.yaml is missing; generated a generic docker run plan.")
		return plan
	}
	if r.Target.Runtime.Native {
		plan.Mode = "native"
		plan.Command = []string{"sh", "-c", "echo native runner is not implemented for this recipe yet"}
		plan.Notes = append(plan.Notes, "Native runner skeleton only. Use recipe artifacts or module installation scripts to install the engine.")
		return plan
	}
	plan.Mode = "unknown"
	plan.Notes = append(plan.Notes, "Recipe does not declare a runnable runtime.")
	return plan
}

func Run(record recipe.Record) (RunPlan, error) {
	plan := BuildRunPlan(record)
	if len(plan.Command) == 0 {
		return plan, nil
	}
	cmd := exec.Command(plan.Command[0], plan.Command[1:]...)
	cmd.Dir = plan.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return plan, cmd.Run()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func portMapping(port int) string {
	if port <= 0 {
		return "8000:8000"
	}
	return strings.TrimSpace(strings.Join([]string{itoa(port), itoa(port)}, ":"))
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
