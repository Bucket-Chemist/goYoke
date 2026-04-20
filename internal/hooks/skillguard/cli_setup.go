package skillguard

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/skillsetup"
)

// cliSetupOutput matches PrepareSkillOutput from the MCP tool for consistency.
type cliSetupOutput struct {
	Skill              string   `json:"skill"`
	TeamDir            string   `json:"team_dir,omitempty"`
	GuardActive        bool     `json:"guard_active"`
	RouterAllowedTools []string `json:"router_allowed_tools,omitempty"`
	Released           bool     `json:"released,omitempty"`
	Error              string   `json:"error,omitempty"`
}

// runSetup handles: goyoke-skill-guard --setup <skill-name> [session-id]
func runSetup() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "[goyoke-skill-guard] ERROR: usage: goyoke-skill-guard --setup <skill-name> [session-id]")
		os.Exit(1)
	}
	skillName := os.Args[2]

	sessionDir := skillsetup.ResolveSessionDir()
	sessionID := ""
	if len(os.Args) >= 4 {
		sessionID = os.Args[3]
	} else {
		sessionID = skillsetup.ResolveSessionID(sessionDir)
	}

	guardConfig, err := skillsetup.LoadSkillGuardConfig(skillName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: load config: %v\n", err)
	}

	if guardConfig == nil {
		emitJSON(cliSetupOutput{Skill: skillName, GuardActive: false})
		return
	}

	// Kill existing lock-holder if a guard with a live PID exists.
	if data, readErr := os.ReadFile(config.GetGuardFilePath(sessionID)); readErr == nil {
		var existing config.ActiveSkill
		if json.Unmarshal(data, &existing) == nil && existing.HolderPID > 0 {
			syscall.Kill(existing.HolderPID, syscall.SIGTERM) //nolint:errcheck
			time.Sleep(100 * time.Millisecond)
		}
	}

	teamDir, err := skillsetup.CreateTeamDir(sessionDir, guardConfig.TeamDirSuffix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] ERROR: create team dir: %v\n", err)
		emitJSON(cliSetupOutput{Skill: skillName, Error: err.Error()})
		os.Exit(1)
	}

	ccPID := os.Getppid()
	lockPath := config.GetGuardLockPath(sessionID)
	holderPID := forkLockHolder(lockPath, ccPID)

	guard := &config.ActiveSkill{
		FormatVersion:      2,
		Skill:              skillName,
		TeamDir:            teamDir,
		RouterAllowedTools: guardConfig.RouterAllowedTools,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
		SessionID:          sessionID,
		HolderPID:          holderPID,
		CCPID:              ccPID,
	}
	if writeErr := skillsetup.WriteGuardFile(guard); writeErr != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: write guard: %v\n", writeErr)
	}

	emitJSON(cliSetupOutput{
		Skill:              skillName,
		TeamDir:            teamDir,
		GuardActive:        true,
		RouterAllowedTools: guardConfig.RouterAllowedTools,
	})
}

// runRelease handles: goyoke-skill-guard --release [session-id]
func runRelease() {
	sessionDir := skillsetup.ResolveSessionDir()
	sessionID := ""
	if len(os.Args) >= 3 {
		sessionID = os.Args[2]
	} else {
		sessionID = skillsetup.ResolveSessionID(sessionDir)
	}

	guardPath := config.GetGuardFilePath(sessionID)
	if data, err := os.ReadFile(guardPath); err == nil {
		var existing config.ActiveSkill
		if json.Unmarshal(data, &existing) == nil && existing.HolderPID > 0 {
			syscall.Kill(existing.HolderPID, syscall.SIGTERM) //nolint:errcheck
			time.Sleep(100 * time.Millisecond)
		}
	}

	if err := skillsetup.RemoveGuardFiles(sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: cleanup: %v\n", err)
	}

	emitJSON(cliSetupOutput{Released: true})
}

// forkLockHolder forks a --hold-lock daemon and returns its PID.
func forkLockHolder(lockPath string, ccPID int) int {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: pipe: %v, continuing unguarded\n", err)
		return 0
	}

	cmd := exec.Command(os.Args[0], "--hold-lock", lockPath, strconv.Itoa(ccPID), "3")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{writePipe}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] Warning: start lock-holder: %v, continuing unguarded\n", err)
		writePipe.Close()
		readPipe.Close()
		return 0
	}
	writePipe.Close()

	ready := make(chan struct{}, 1)
	go func() {
		buf := make([]byte, 1)
		readPipe.Read(buf) //nolint:errcheck
		ready <- struct{}{}
	}()

	holderPID := cmd.Process.Pid
	select {
	case <-ready:
		// Lock holder ready.
	case <-time.After(2 * time.Second):
		fmt.Fprintln(os.Stderr, "[goyoke-skill-guard] Warning: lock-holder readiness timeout, continuing unguarded")
		holderPID = 0
	}
	readPipe.Close()
	cmd.Process.Release() //nolint:errcheck

	return holderPID
}

func emitJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-skill-guard] ERROR: marshal output: %v\n", err)
		fmt.Println("{}")
		return
	}
	fmt.Println(string(data))
}
