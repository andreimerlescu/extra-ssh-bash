package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/andreimerlescu/configurable"
	"github.com/andreimerlescu/extra-ssh-bash/cmd/command"
	sema "github.com/andreimerlescu/go-sema"
)

type config struct {
	ctx         context.Context
	cfg         configurable.IConfigurable
	limit       sema.Semaphore
	api         *string
	projectId   *int
	json        *bool
	tfDir       *string
	key         *string
	user        *string
	accessToken *string
	bash        *string
	stderr      *string
	stdout      *string
	ipCSV       *string
	tfOutputVar *string
}

func commonValidator(co command.CommandOutput) bool {
	//log.Printf("COMMAND = %s", co.Command)
	//log.Printf("STDOUT = %s", co.Stdout)
	//log.Printf("STDERR = %s", co.Stderr)
	//log.Printf("ERROR = %s", co.Error)
	return len(co.Stderr) == 0 && len(co.Stdout) > 0
}

const defaultTerraformState = "default-tfstate"

func (c *config) terraformStateName() string {
	dirInfo, dirErr := os.Lstat(*c.tfDir)
	if dirErr != nil {
		log.Printf("terraformStateName() dirErr = %v", dirErr)
		return defaultTerraformState
	}
	return fmt.Sprintf("%s-tfstate", dirInfo.Name())
}

func (c *config) getEnv() []string {
	stateName := c.terraformStateName()
	address := fmt.Sprintf("%s/projects/%d/terraform/state/%s", *c.api, *c.projectId, stateName)
	env := os.Environ()
	env = append(env, fmt.Sprintf("%s=%s", "TF_STATE_NAME", stateName))
	env = append(env, fmt.Sprintf("%s=%s", "TF_HTTP_USERNAME", *c.user))
	env = append(env, fmt.Sprintf("%s=%s", "TF_HTTP_PASSWORD", *c.accessToken))
	env = append(env, fmt.Sprintf("%s=%s", "TF_HTTP_ADDRESS", address))
	env = append(env, fmt.Sprintf("%s=%s", "TF_HTTP_LOCK_ADDRESS", fmt.Sprintf("%s/lock", address)))
	env = append(env, fmt.Sprintf("%s=%s", "TF_HTTP_UNLOCK_ADDRESS", fmt.Sprintf("%s/unlock", address)))
	env = append(env, fmt.Sprintf("%s=%s", "TF_HTTP_LOCK_METHOD", `"POST"`))
	env = append(env, fmt.Sprintf("%s=%s", "TF_HTTP_UNLOCK_METHOD", `"DELETE"`))
	env = append(env, fmt.Sprintf("%s=%s", "TF_HTTP_RETRY_WAIT_MIN", `5`))
	return env
}

func (c *config) terraformPublicIPs() []string {
	var cmd string
	if *c.tfOutputVar == "public_ips" {
		// use JSON for public_ips
		cmd = fmt.Sprintf("%s=%s %s %s", "terraform -chdir", *c.tfDir, "output -json", *c.tfOutputVar)
	} else if *c.tfOutputVar == "public_ip" {
		cmd = fmt.Sprintf("%s=%s %s %s", "terraform -chdir", *c.tfDir, "output", *c.tfOutputVar)
	} else {
		log.Fatalf("Cannot use --tfoutputvar=%s here. Valid options are: public_ip, public_ips", *c.tfOutputVar)
	}
	cmdOutput, cmdOk := command.Prompt().RunInside(c.ctx, cmd, c.limit, *c.tfDir, c.getEnv(), commonValidator)
	if !cmdOk {
		log.Println(cmd)
		log.Printf("terraformPublicIPs() cmdOutput !ok\n\nSTDERR = %s\n\nSTDOUT = %s\n", cmdOutput.Stderr, cmdOutput.Stdout)
		return []string{}
	}
	target := []string{}
	jsonErr := json.Unmarshal(cmdOutput.Stdout, &target)
	if jsonErr != nil {
		log.Fatalln(jsonErr)
	}
	return target
}

func (c *config) isUsingTerraform() bool {
	dirInfo, dirErr := os.Lstat(*c.tfDir)
	if dirErr != nil {
		log.Printf("isUsingTerraform() dirErr = %v", dirErr)
		return false
	}
	if !dirInfo.IsDir() {
		log.Printf("isUsingTerraform() --tfdir is not a directory and must be")
		return false
	}
	return len(*c.ipCSV) == 0
}

func (c *config) Parse() error {
	configFile := filepath.Join(".", "config.yaml")
	_, statErr := os.Stat(configFile)
	if statErr != nil {
		cfgErr := c.cfg.Parse("")
		if cfgErr != nil {
			return cfgErr
		}
	} else {
		cfgErr := c.cfg.Parse(configFile)
		if cfgErr != nil {
			return cfgErr
		}
	}
	return nil
}

type application struct {
	ctx    context.Context
	cfg    configurable.IConfigurable
	config config
	limit  sema.Semaphore
}

func main() {
	// Create a CLI application
	var app application = application{
		ctx:   context.Background(),
		cfg:   configurable.New(),
		limit: sema.New(runtime.GOMAXPROCS(0)),
	}

	// Define arguments
	app.config = config{
		ctx:         app.ctx,
		cfg:         app.cfg,
		limit:       app.limit,
		api:         app.cfg.NewString("api", "https://gitlab.com/api/v4", "GitLab API URL"),
		projectId:   app.cfg.NewInt("id", 1, "GitLab Project ID"),
		json:        app.cfg.NewBool("json", false, "Use JSON formatted output"),
		user:        app.cfg.NewString("user", "ubuntu", "Username of remote host"),
		key:         app.cfg.NewString("key", filepath.Join(".", ".ssh", "id_ed25519"), "Path to SSH key for remote access"),
		tfDir:       app.cfg.NewString("tfdir", filepath.Join(".", "terraform"), "Path to terraform directory"),
		bash:        app.cfg.NewString("bash", "", "Bash command to execute remotely"),
		stdout:      app.cfg.NewString("stdout", filepath.Join(".", "logs", "go.ebs.stdout"), "Path to STDOUT to write to"),
		stderr:      app.cfg.NewString("stderr", filepath.Join(".", "logs", "go.ebs.stderr"), "Path to STDERR to write to"),
		ipCSV:       app.cfg.NewString("ipcsv", "", "CSV string of IP addresses"),
		accessToken: app.cfg.NewString("token", "", "GitLab API Access Token"),
		tfOutputVar: app.cfg.NewString("tfoutputvar", "public_ips", "Output variable name from Terraform to get IP addresses of target hosts"),
	}

	// Parse arguments and/or config.yaml
	cfgErr := app.config.Parse()
	if cfgErr != nil {
		log.Fatalln(cfgErr)
	}

	var ips []string
	sshOpts := "-o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o CheckHostIP=no"
	wg := sync.WaitGroup{}

	type Result struct {
		Cmd    string `json:"cmd"`
		Stdout string `json:"stdout"`
		Stderr string `json:"stderr"`
	}
	results := make(map[string]Result)

	// Validations
	if app.config.isUsingTerraform() {
		ips = app.config.terraformPublicIPs()
		for _, ip := range ips {
			wg.Add(1)
			go func(ctx context.Context, wg *sync.WaitGroup, ip string, limit sema.Semaphore) {
				defer wg.Done()
				cmd := fmt.Sprintf("%s %s %s %s@%s %s", "ssh -i", *app.config.key, sshOpts, *app.config.user, ip, *app.config.bash)
				output, ok := command.Prompt().RunInside(ctx, cmd, limit, *app.config.tfDir, app.config.getEnv(), commonValidator)
				if !ok {
					log.Printf("failed to exec cmd:\n\n%s\n\nSTDOUT = %s\nSTDERR = %s\n\n", cmd, output.Stdout, output.Stderr)
				}
				results[ip] = Result{
					Cmd:    cmd,
					Stdout: string(output.Stdout),
					Stderr: string(output.Stderr),
				}
			}(app.ctx, &wg, ip, app.limit)
		}
		wg.Wait()
	} else {
		*app.config.ipCSV = strings.ReplaceAll(*app.config.ipCSV, " ", "")
		ips = strings.Split(*app.config.ipCSV, ",")
		for _, ip := range ips {
			wg.Add(1)
			go func(ctx context.Context, wg *sync.WaitGroup, ip string, limit sema.Semaphore) {
				defer wg.Done()
				cmd := fmt.Sprintf("%s %s %s %s@%s %s", "ssh -i", *app.config.key, sshOpts, *app.config.user, ip, *app.config.bash)
				output, ok := command.Prompt().RunInside(ctx, cmd, limit, *app.config.tfDir, app.config.getEnv(), commonValidator)
				if !ok {
					log.Printf("failed to exec cmd:\n\n%s\n\nSTDOUT = %s\nSTDERR = %s\n\n", cmd, output.Stdout, output.Stderr)
				}
				results[ip] = Result{
					Cmd:    cmd,
					Stdout: string(output.Stdout),
					Stderr: string(output.Stderr),
				}
			}(app.ctx, &wg, ip, app.limit)
		}
		wg.Wait()
	}

	if !*app.config.json {
		for ip, result := range results {
			_, _ = fmt.Fprintf(os.Stdout, "Host %s:\n---------------------\nCommand: %s\n\n%s\n", ip, result.Cmd, string(result.Stdout))
		}
	} else {
		bytes, err := json.Marshal(results)
		if err != nil {
			log.Fatalln(err)
		}
		_, _ = fmt.Fprintf(os.Stdout, "%s", string(bytes))
	}

}
