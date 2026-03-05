// ClawDroid - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 ClawDroid contributors

package main

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/KarakuriAgent/clawdroid/pkg/agent"
	"github.com/KarakuriAgent/clawdroid/pkg/bus"
	"github.com/KarakuriAgent/clawdroid/pkg/channels"
	"github.com/KarakuriAgent/clawdroid/pkg/config"
	"github.com/KarakuriAgent/clawdroid/pkg/cron"
	"github.com/KarakuriAgent/clawdroid/pkg/gateway"
	"github.com/KarakuriAgent/clawdroid/pkg/heartbeat"
	"github.com/KarakuriAgent/clawdroid/pkg/logger"
	"github.com/KarakuriAgent/clawdroid/pkg/providers"
	"github.com/KarakuriAgent/clawdroid/pkg/skills"
	"github.com/KarakuriAgent/clawdroid/pkg/tools"
	_ "time/tzdata"

	"github.com/chzyer/readline"
)

func init() {
	if tz := os.Getenv("TZ"); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			time.Local = loc
		}
	}
}

//go:generate cp -r ../../workspace .
//go:embed workspace
var embeddedFiles embed.FS

var (
	version   = "dev"
	gitCommit string
	buildTime string
	goVersion string
)

const logo = "🦞"

// formatVersion returns the version string with optional git commit
func formatVersion() string {
	v := version
	if gitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", gitCommit)
	}
	return v
}

// formatBuildInfo returns build time and go version info
func formatBuildInfo() (build string, goVer string) {
	if buildTime != "" {
		build = buildTime
	}
	goVer = goVersion
	if goVer == "" {
		goVer = runtime.Version()
	}
	return
}

func printVersion() {
	fmt.Printf("%s clawdroid %s\n", logo, formatVersion())
	build, goVer := formatBuildInfo()
	if build != "" {
		fmt.Printf("  Build: %s\n", build)
	}
	if goVer != "" {
		fmt.Printf("  Go: %s\n", goVer)
	}
}

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = srcFile.Close() }()

		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer func() { _ = dstFile.Close() }()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}

func init() {
	// Override DNS resolver for CGO_ENABLED=0 builds (Android APK).
	// The pure-Go resolver falls back to [::1]:53 on Android because
	// /etc/resolv.conf does not exist. When CGO is enabled (Termux),
	// the cgo resolver is used instead and this Dial is never called.
	net.DefaultResolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "udp", "8.8.8.8:53")
	}

	// Set CA certificate location for CGO_ENABLED=0 builds (Android APK).
	// The pure-Go TLS stack cannot find Android's CA certs at their
	// non-standard path. Only set if not already configured.
	if os.Getenv("SSL_CERT_FILE") == "" && os.Getenv("SSL_CERT_DIR") == "" {
		if _, err := os.Stat("/system/etc/security/cacerts"); err == nil {
			_ = os.Setenv("SSL_CERT_DIR", "/system/etc/security/cacerts")
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "onboard":
		onboard()
	case "agent":
		agentCmd()
	case "gateway":
		gatewayCmd()
	case "status":
		statusCmd()
	case "cron":
		cronCmd()
	case "skills":
		if len(os.Args) < 3 {
			skillsHelp()
			return
		}

		subcommand := os.Args[2]

		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		dataDir := cfg.DataPath()
		// 获取全局配置目录和内置 skills 目录
		globalDir := filepath.Dir(getConfigPath())
		globalSkillsDir := filepath.Join(globalDir, "skills")
		builtinSkillsDir := filepath.Join(globalDir, "clawdroid", "skills")
		skillsLoader := skills.NewSkillsLoader(dataDir, globalSkillsDir, builtinSkillsDir)

		switch subcommand {
		case "list":
			skillsListCmd(skillsLoader)
		case "remove", "uninstall":
			if len(os.Args) < 4 {
				fmt.Println("Usage: clawdroid skills remove <skill-name>")
				return
			}
			skillsRemoveCmd(dataDir, os.Args[3])
		case "install-builtin":
			skillsInstallBuiltinCmd(dataDir)
		case "list-builtin":
			skillsListBuiltinCmd()
		case "show":
			if len(os.Args) < 4 {
				fmt.Println("Usage: clawdroid skills show <skill-name>")
				return
			}
			skillsShowCmd(skillsLoader, os.Args[3])
		default:
			fmt.Printf("Unknown skills command: %s\n", subcommand)
			skillsHelp()
		}
	case "version", "--version", "-v":
		printVersion()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf("%s clawdroid - Personal AI Assistant v%s\n\n", logo, version)
	fmt.Println("Usage: clawdroid <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  onboard     Initialize clawdroid configuration and workspace")
	fmt.Println("  agent       Interact with the agent directly")
	fmt.Println("  gateway     Start clawdroid gateway")
	fmt.Println("  status      Show clawdroid status")
	fmt.Println("  cron        Manage scheduled tasks")
	fmt.Println("  skills      Manage skills (install, list, remove)")
	fmt.Println("  version     Show version information")
}

func onboard() {
	configPath := getConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists at %s\n", configPath)
		fmt.Print("Overwrite? (y/n): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" {
			fmt.Println("Aborted.")
			return
		}
	}

	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()
	dataDir := cfg.DataPath()
	_ = os.MkdirAll(workspace, 0755)
	createWorkspaceTemplates(dataDir)

	fmt.Printf("%s clawdroid is ready!\n", logo)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add your API key to", configPath)
	fmt.Println("     Get one at: https://openrouter.ai/keys")
	fmt.Println("  2. Chat: clawdroid agent -m \"Hello!\"")
}

func copyEmbeddedToTarget(targetDir string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Walk through all files in embed.FS
	err := fs.WalkDir(embeddedFiles, "workspace", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read embedded file
		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		new_path, err := filepath.Rel("workspace", path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		// Build target file path
		targetPath := filepath.Join(targetDir, new_path)

		// Ensure target file's directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(targetPath), err)
		}

		// Write file
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		return nil
	})

	return err
}

func createWorkspaceTemplates(workspace string) {
	err := copyEmbeddedToTarget(workspace)
	if err != nil {
		fmt.Printf("Error copying workspace templates: %v\n", err)
	}
}

func agentCmd() {
	message := ""
	sessionKey := "cli:default"

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--debug", "-d":
			logger.SetLevel(logger.DEBUG)
			fmt.Println("🔍 Debug mode enabled")
		case "-m", "--message":
			if i+1 < len(args) {
				message = args[i+1]
				i++
			}
		case "-s", "--session":
			if i+1 < len(args) {
				sessionKey = args[i+1]
				i++
			}
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	provider, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Printf("Error creating provider: %v\n", err)
		os.Exit(1)
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Print agent startup info (only for interactive mode)
	startupInfo := agentLoop.GetStartupInfo()
	logger.InfoCF("agent", "Agent initialized",
		map[string]interface{}{
			"tools_count":      startupInfo["tools"].(map[string]interface{})["count"],
			"skills_total":     startupInfo["skills"].(map[string]interface{})["total"],
			"skills_available": startupInfo["skills"].(map[string]interface{})["available"],
		})

	if message != "" {
		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, message, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n%s %s\n", logo, response)
	} else {
		fmt.Printf("%s Interactive mode (Ctrl+C to exit)\n\n", logo)
		interactiveMode(agentLoop, sessionKey)
	}
}

func interactiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	prompt := fmt.Sprintf("%s You: ", logo)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     filepath.Join(os.TempDir(), ".clawdroid_history"),
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})

	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		fmt.Println("Falling back to simple input mode...")
		simpleInteractiveMode(agentLoop, sessionKey)
		return
	}
	defer func() { _ = rl.Close() }()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", logo, response)
	}
}

func simpleInteractiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s You: ", logo)
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", logo, response)
	}
}

func gatewayCmd() {
	// Check for --debug flag
	args := os.Args[2:]
	for _, arg := range args {
		if arg == "--debug" || arg == "-d" {
			logger.SetLevel(logger.DEBUG)
			fmt.Println("🔍 Debug mode enabled")
			break
		}
	}

	configPath := getConfigPath()
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// If config.json does not exist, start in setup mode (minimal gateway)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		gatewaySetupMode(cfg, configPath)
		return
	}

	msgBus := bus.NewMessageBus()

	// Restart channel for config-triggered restarts
	restartCh := make(chan struct{}, 1)

	// Start Gateway HTTP server (Config API) first — must be available
	// even when LLM provider fails, so the user can fix config via API.
	gwServer := gateway.NewServer(cfg, configPath, func() {
		select {
		case restartCh <- struct{}{}:
		default:
		}
	})
	if err := gwServer.Start(); err != nil {
		fmt.Printf("Error starting gateway HTTP server: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Config API started on 127.0.0.1:%d\n", cfg.Gateway.Port)

	channelManager, err := channels.NewManager(cfg, msgBus, configPath)
	if err != nil {
		fmt.Printf("Error creating channel manager: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := channelManager.StartAll(ctx); err != nil {
		fmt.Printf("Error starting channels: %v\n", err)
	}

	enabledChannels := channelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("✓ Channels enabled: %s\n", enabledChannels)
	} else {
		fmt.Println("⚠ Warning: No channels enabled")
	}

	// Try to create LLM provider — if it fails, run in degraded mode
	// (Gateway + Channels available, but no AgentLoop).
	provider, providerErr := providers.CreateProvider(cfg)

	var agentLoop *agent.AgentLoop
	var cronService *cron.CronService
	var heartbeatService *heartbeat.HeartbeatService

	if providerErr != nil {
		fmt.Printf("⚠ LLM provider not available: %v\n", providerErr)
		fmt.Println("  → Running in degraded mode. Fix LLM settings via Config API, then restart.")

		// Drain inbound messages and reply with an error
		go func() {
			for {
				msg, ok := msgBus.ConsumeInbound(ctx)
				if !ok {
					return
				}
				msgBus.PublishOutbound(bus.OutboundMessage{
					Channel: msg.Channel,
					ChatID:  msg.ChatID,
					Content: "⚠ LLM is not configured. Please set your model and API key in Settings, then restart the gateway.",
				})
			}
		}()
	} else {
		agentLoop = agent.NewAgentLoop(cfg, msgBus, provider)

		// Print agent startup info
		fmt.Println("\n📦 Agent Status:")
		startupInfo := agentLoop.GetStartupInfo()
		toolsInfo := startupInfo["tools"].(map[string]interface{})
		skillsInfo := startupInfo["skills"].(map[string]interface{})
		fmt.Printf("  • Tools: %d loaded\n", toolsInfo["count"])
		fmt.Printf("  • Skills: %d/%d available\n",
			skillsInfo["available"],
			skillsInfo["total"])

		logger.InfoCF("agent", "Agent initialized",
			map[string]interface{}{
				"tools_count":      toolsInfo["count"],
				"skills_total":     skillsInfo["total"],
				"skills_available": skillsInfo["available"],
			})

		agentLoop.SetChannelManager(channelManager)

		cronService = setupCronTool(agentLoop, msgBus, cfg.WorkspacePath(), cfg.DataPath(), cfg.Agents.Defaults.RestrictToWorkspace, cfg.Tools.Exec.Enabled)

		heartbeatService = heartbeat.NewHeartbeatService(
			cfg.WorkspacePath(),
			cfg.DataPath(),
			cfg.Heartbeat.Interval,
			cfg.Heartbeat.Enabled,
			agentLoop.StateManager(),
		)
		heartbeatService.SetBus(msgBus)
		heartbeatService.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
			if channel == "" || chatID == "" {
				channel, chatID = "cli", "direct"
			}
			response, err := agentLoop.ProcessHeartbeat(context.Background(), prompt, channel, chatID)
			if err != nil {
				return tools.ErrorResult(fmt.Sprintf("Heartbeat error: %v", err))
			}
			if response == "HEARTBEAT_OK" {
				return tools.SilentResult("Heartbeat OK")
			}
			return tools.SilentResult(response)
		})

		if err := cronService.Start(); err != nil {
			fmt.Printf("Error starting cron service: %v\n", err)
		}
		fmt.Println("✓ Cron service started")

		if err := heartbeatService.Start(); err != nil {
			fmt.Printf("Error starting heartbeat service: %v\n", err)
		}
		fmt.Println("✓ Heartbeat service started")

		go func() { _ = agentLoop.Run(ctx) }()
	}

	fmt.Printf("✓ Gateway started on 127.0.0.1:%d\n", cfg.Gateway.Port)
	fmt.Println("Press Ctrl+C to stop")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	restart := false
	select {
	case <-sigChan:
	case <-restartCh:
		restart = true
		fmt.Println("\nRestarting due to config change...")
	}

	// Graceful shutdown
	fmt.Println("\nShutting down...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = gwServer.Stop(shutdownCtx)
	if heartbeatService != nil {
		heartbeatService.Stop()
	}
	if cronService != nil {
		cronService.Stop()
	}
	if agentLoop != nil {
		agentLoop.Stop()
	}
	_ = channelManager.StopAll(shutdownCtx)
	fmt.Println("✓ Gateway stopped")

	if restart {
		execRestart()
	}
}

func execRestart() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("Error finding executable: %v\n", err)
		os.Exit(1)
	}
	if err := syscall.Exec(exe, os.Args, os.Environ()); err != nil {
		fmt.Printf("Error restarting: %v\n", err)
		os.Exit(1)
	}
}

// gatewaySetupMode starts a minimal gateway with only HTTP + WebSocket.
// Provider, AgentLoop, CronService, and HeartbeatService are not started.
func gatewaySetupMode(cfg *config.Config, configPath string) {
	fmt.Println("\n⚙ Starting in setup mode (config.json not found)")

	restartCh := make(chan struct{}, 1)

	gwServer := gateway.NewServer(cfg, configPath, func() {
		select {
		case restartCh <- struct{}{}:
		default:
		}
	})
	if err := gwServer.Start(); err != nil {
		fmt.Printf("Error starting gateway HTTP server: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Config API started on 127.0.0.1:%d\n", cfg.Gateway.Port)

	msgBus := bus.NewMessageBus()
	channelManager, err := channels.NewManager(cfg, msgBus, configPath)
	if err != nil {
		fmt.Printf("Error creating channel manager: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := channelManager.StartAll(ctx); err != nil {
		fmt.Printf("Error starting channels: %v\n", err)
	}

	enabledChannels := channelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("✓ Channels enabled: %s\n", enabledChannels)
	}

	fmt.Println("✓ Setup mode ready — waiting for setup wizard")
	fmt.Println("Press Ctrl+C to stop")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	restart := false
	select {
	case <-sigChan:
	case <-restartCh:
		restart = true
		fmt.Println("\nRestarting after setup complete...")
	}

	fmt.Println("\nShutting down...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = gwServer.Stop(shutdownCtx)
	_ = channelManager.StopAll(shutdownCtx)
	fmt.Println("✓ Gateway stopped")

	if restart {
		execRestart()
	}
}

func statusCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	configPath := getConfigPath()

	fmt.Printf("%s clawdroid Status\n", logo)
	fmt.Printf("Version: %s\n", formatVersion())
	build, _ := formatBuildInfo()
	if build != "" {
		fmt.Printf("Build: %s\n", build)
	}
	fmt.Println()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("Config:", configPath, "✓")
	} else {
		fmt.Println("Config:", configPath, "✗")
	}

	workspace := cfg.WorkspacePath()
	if _, err := os.Stat(workspace); err == nil {
		fmt.Println("Workspace:", workspace, "✓")
	} else {
		fmt.Println("Workspace:", workspace, "✗")
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Model: %s\n", cfg.LLM.Model)
		if cfg.LLM.APIKey != "" {
			fmt.Println("API Key: ✓")
		} else {
			fmt.Println("API Key: not set")
		}
		if cfg.LLM.BaseURL != "" {
			fmt.Printf("Base URL: %s\n", cfg.LLM.BaseURL)
		}
	}
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clawdroid", "config.json")
}

func setupCronTool(agentLoop *agent.AgentLoop, msgBus *bus.MessageBus, workspace string, dataDir string, restrict bool, execEnabled bool) *cron.CronService {
	cronStorePath := filepath.Join(dataDir, "cron", "jobs.json")

	// Create cron service
	cronService := cron.NewCronService(cronStorePath, nil)

	// Create and register CronTool (workspace is for ExecTool sandboxing)
	cronTool := tools.NewCronTool(cronService, agentLoop, msgBus, workspace, restrict, execEnabled)
	agentLoop.RegisterTool(cronTool)

	// Set the onJob handler
	cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
		result := cronTool.ExecuteJob(context.Background(), job)
		return result, nil
	})

	return cronService
}

func loadConfig() (*config.Config, error) {
	return config.LoadConfig(getConfigPath())
}

func cronCmd() {
	if len(os.Args) < 3 {
		cronHelp()
		return
	}

	subcommand := os.Args[2]

	// Load config to get workspace path
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	cronStorePath := filepath.Join(cfg.DataPath(), "cron", "jobs.json")

	switch subcommand {
	case "list":
		cronListCmd(cronStorePath)
	case "add":
		cronAddCmd(cronStorePath)
	case "remove":
		if len(os.Args) < 4 {
			fmt.Println("Usage: clawdroid cron remove <job_id>")
			return
		}
		cronRemoveCmd(cronStorePath, os.Args[3])
	case "enable":
		cronEnableCmd(cronStorePath, false)
	case "disable":
		cronEnableCmd(cronStorePath, true)
	default:
		fmt.Printf("Unknown cron command: %s\n", subcommand)
		cronHelp()
	}
}

func cronHelp() {
	fmt.Println("\nCron commands:")
	fmt.Println("  list              List all scheduled jobs")
	fmt.Println("  add              Add a new scheduled job")
	fmt.Println("  remove <id>       Remove a job by ID")
	fmt.Println("  enable <id>      Enable a job")
	fmt.Println("  disable <id>     Disable a job")
	fmt.Println()
	fmt.Println("Add options:")
	fmt.Println("  -n, --name       Job name")
	fmt.Println("  -m, --message    Message for agent")
	fmt.Println("  -e, --every      Run every N seconds")
	fmt.Println("  -c, --cron       Cron expression (e.g. '0 9 * * *')")
	fmt.Println("  -d, --deliver     Deliver response to channel")
	fmt.Println("  --to             Recipient for delivery")
	fmt.Println("  --channel        Channel for delivery")
}

func cronListCmd(storePath string) {
	cs := cron.NewCronService(storePath, nil)
	jobs := cs.ListJobs(true) // Show all jobs, including disabled

	if len(jobs) == 0 {
		fmt.Println("No scheduled jobs.")
		return
	}

	fmt.Println("\nScheduled Jobs:")
	fmt.Println("----------------")
	for _, job := range jobs {
		var schedule string
		if job.Schedule.Kind == "every" && job.Schedule.EveryMS != nil {
			schedule = fmt.Sprintf("every %ds", *job.Schedule.EveryMS/1000)
		} else if job.Schedule.Kind == "cron" {
			schedule = job.Schedule.Expr
		} else {
			schedule = "one-time"
		}

		nextRun := "scheduled"
		if job.State.NextRunAtMS != nil {
			nextTime := time.UnixMilli(*job.State.NextRunAtMS)
			nextRun = nextTime.Format("2006-01-02 15:04")
		}

		status := "enabled"
		if !job.Enabled {
			status = "disabled"
		}

		fmt.Printf("  %s (%s)\n", job.Name, job.ID)
		fmt.Printf("    Schedule: %s\n", schedule)
		fmt.Printf("    Status: %s\n", status)
		fmt.Printf("    Next run: %s\n", nextRun)
	}
}

func cronAddCmd(storePath string) {
	name := ""
	message := ""
	var everySec *int64
	cronExpr := ""
	deliver := false
	channel := ""
	to := ""

	args := os.Args[3:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--name":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "-m", "--message":
			if i+1 < len(args) {
				message = args[i+1]
				i++
			}
		case "-e", "--every":
			if i+1 < len(args) {
				var sec int64
				_, _ = fmt.Sscanf(args[i+1], "%d", &sec)
				everySec = &sec
				i++
			}
		case "-c", "--cron":
			if i+1 < len(args) {
				cronExpr = args[i+1]
				i++
			}
		case "-d", "--deliver":
			deliver = true
		case "--to":
			if i+1 < len(args) {
				to = args[i+1]
				i++
			}
		case "--channel":
			if i+1 < len(args) {
				channel = args[i+1]
				i++
			}
		}
	}

	if name == "" {
		fmt.Println("Error: --name is required")
		return
	}

	if message == "" {
		fmt.Println("Error: --message is required")
		return
	}

	if everySec == nil && cronExpr == "" {
		fmt.Println("Error: Either --every or --cron must be specified")
		return
	}

	var schedule cron.CronSchedule
	if everySec != nil {
		everyMS := *everySec * 1000
		schedule = cron.CronSchedule{
			Kind:    "every",
			EveryMS: &everyMS,
		}
	} else {
		schedule = cron.CronSchedule{
			Kind: "cron",
			Expr: cronExpr,
		}
	}

	cs := cron.NewCronService(storePath, nil)
	job, err := cs.AddJob(name, schedule, message, deliver, channel, to)
	if err != nil {
		fmt.Printf("Error adding job: %v\n", err)
		return
	}

	fmt.Printf("✓ Added job '%s' (%s)\n", job.Name, job.ID)
}

func cronRemoveCmd(storePath, jobID string) {
	cs := cron.NewCronService(storePath, nil)
	if cs.RemoveJob(jobID) {
		fmt.Printf("✓ Removed job %s\n", jobID)
	} else {
		fmt.Printf("✗ Job %s not found\n", jobID)
	}
}

func cronEnableCmd(storePath string, disable bool) {
	if len(os.Args) < 4 {
		fmt.Println("Usage: clawdroid cron enable/disable <job_id>")
		return
	}

	jobID := os.Args[3]
	cs := cron.NewCronService(storePath, nil)
	enabled := !disable

	job := cs.EnableJob(jobID, enabled)
	if job != nil {
		status := "enabled"
		if disable {
			status = "disabled"
		}
		fmt.Printf("✓ Job '%s' %s\n", job.Name, status)
	} else {
		fmt.Printf("✗ Job %s not found\n", jobID)
	}
}

func skillsHelp() {
	fmt.Println("\nSkills commands:")
	fmt.Println("  list                    List installed skills")
	fmt.Println("  install-builtin         Install all builtin skills to workspace")
	fmt.Println("  list-builtin            List available builtin skills")
	fmt.Println("  remove <name>           Remove installed skill")
	fmt.Println("  show <name>             Show skill details")
	fmt.Println()
	fmt.Println("To install custom skills, place SKILL.md files manually in:")
	fmt.Println("  ~/.clawdroid/data/skills/<skill-name>/SKILL.md")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  clawdroid skills list")
	fmt.Println("  clawdroid skills install-builtin")
	fmt.Println("  clawdroid skills list-builtin")
	fmt.Println("  clawdroid skills remove <name>")
}

func skillsListCmd(loader *skills.SkillsLoader) {
	allSkills := loader.ListSkills()

	if len(allSkills) == 0 {
		fmt.Println("No skills installed.")
		return
	}

	fmt.Println("\nInstalled Skills:")
	fmt.Println("------------------")
	for _, skill := range allSkills {
		fmt.Printf("  ✓ %s (%s)\n", skill.Name, skill.Source)
		if skill.Description != "" {
			fmt.Printf("    %s\n", skill.Description)
		}
	}
}

func skillsRemoveCmd(dataDir string, skillName string) {
	skillDir := filepath.Join(dataDir, "skills", skillName)

	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		fmt.Printf("✗ Skill '%s' not found\n", skillName)
		os.Exit(1)
	}

	fmt.Printf("Removing skill '%s'...\n", skillName)
	if err := os.RemoveAll(skillDir); err != nil {
		fmt.Printf("✗ Failed to remove skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Skill '%s' removed successfully!\n", skillName)
}

func skillsInstallBuiltinCmd(dataDir string) {
	builtinSkillsDir := "./clawdroid/skills"
	skillsDir := filepath.Join(dataDir, "skills")

	fmt.Printf("Copying builtin skills to workspace...\n")

	skillsToInstall := []string{}

	for _, skillName := range skillsToInstall {
		builtinPath := filepath.Join(builtinSkillsDir, skillName)
		destPath := filepath.Join(skillsDir, skillName)

		if _, err := os.Stat(builtinPath); err != nil {
			fmt.Printf("⊘ Builtin skill '%s' not found: %v\n", skillName, err)
			continue
		}

		if err := os.MkdirAll(destPath, 0755); err != nil {
			fmt.Printf("✗ Failed to create directory for %s: %v\n", skillName, err)
			continue
		}

		if err := copyDirectory(builtinPath, destPath); err != nil {
			fmt.Printf("✗ Failed to copy %s: %v\n", skillName, err)
		}
	}

	fmt.Println("\n✓ All builtin skills installed!")
	fmt.Println("Now you can use them in your workspace.")
}

func skillsListBuiltinCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	builtinSkillsDir := filepath.Join(filepath.Dir(cfg.WorkspacePath()), "clawdroid", "skills")

	fmt.Println("\nAvailable Builtin Skills:")
	fmt.Println("-----------------------")

	entries, err := os.ReadDir(builtinSkillsDir)
	if err != nil {
		fmt.Printf("Error reading builtin skills: %v\n", err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No builtin skills available.")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			skillName := entry.Name()
			skillFile := filepath.Join(builtinSkillsDir, skillName, "SKILL.md")

			description := "No description"
			if _, err := os.Stat(skillFile); err == nil {
				data, err := os.ReadFile(skillFile)
				if err == nil {
					content := string(data)
					if idx := strings.Index(content, "\n"); idx > 0 {
						firstLine := content[:idx]
						if strings.Contains(firstLine, "description:") {
							descLine := strings.Index(content[idx:], "\n")
							if descLine > 0 {
								description = strings.TrimSpace(content[idx+descLine : idx+descLine])
							}
						}
					}
				}
			}
			status := "✓"
			fmt.Printf("  %s  %s\n", status, entry.Name())
			if description != "" {
				fmt.Printf("     %s\n", description)
			}
		}
	}
}

func skillsShowCmd(loader *skills.SkillsLoader, skillName string) {
	content, ok := loader.LoadSkill(skillName)
	if !ok {
		fmt.Printf("✗ Skill '%s' not found\n", skillName)
		return
	}

	fmt.Printf("\n📦 Skill: %s\n", skillName)
	fmt.Println("----------------------")
	fmt.Println(content)
}
