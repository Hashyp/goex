# Agent Development Guide - defaultdevcontainer

You are Go expert designing and implementing high quality, clean code solution which favours SOLID principles

## Project Overview

defaultdevcontainer is a terminal-based dual-pane file browser built with bubbletea that allows navigation and file operations between local file systems and Azure Blob Storage.

**Technology Stack:**
- Go 1.25
- "github.com/charmbracelet/bubbles/table"
- "github.com/charmbracelet/bubbletea"
- "github.com/charmbracelet/lipgloss"

## 🚨 CRITICAL: Container Requirements

**ALL PROJECT COMMANDS MUST BE RUN INSIDE THE DEVCONTAINER, EXCEPT GIT/GH., creating new files, reading existing files and writing to files**

This project uses a devcontainer environment with all dependencies pre-configured. The working directory inside the container is `/workspaces/defaultdevcontainer`.

### Required: Execute All Commands Inside Container

Before running ANY command in this project, you MUST execute them inside the devcontainer using `docker exec`.
The only exception is Git and GitHub CLI commands, which can be run outside the container:

```bash
# Most commands MUST be prefixed with:
docker exec default-go-devcontainer <command>

# Or open an interactive bash shell:
docker exec -it default-go-devcontainer bash

# Git and GitHub CLI can run outside the container:
git status
gh pr create
```

**DO NOT** run non-git/gh commands outside the container - they will fail because:
- `go` is only available inside the container
- Azurite service runs only in the container
- Azure Storage SDK configuration is container-specific
- Working directory is `/workspaces/defaultdevcontainer` inside container

## 🚨 CRITICAL: Branching and Clean Working Tree

**ALWAYS start a new feature branch before any implementation work.**

Before starting a feature:
- Ensure the working tree is clean (no unstaged or uncommitted files).
- Switch to `main` successfully.
- Create a new feature branch from `main`.

This ensures work is isolated and you can always return to `main` cleanly.

## ✅ Definition of Done for Features

**A feature is only considered done after tests have run successfully.**

**Always run the full test suite after fixing a broken feature.**

**Provide a concise commit message summary of the work done when asked.**

After implementing any feature, always run the test suite using the container command:
```bash
```

