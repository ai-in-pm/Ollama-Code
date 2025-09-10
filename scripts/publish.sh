#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   scripts/publish.sh https://github.com/ai-in-pm/Ollama-Code.git
# If no URL is passed, it will default to the official repo URL below.

REPO_URL="${1:-https://github.com/ai-in-pm/Ollama-Code.git}"

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [ -d .git ]; then
  echo "Git repo already exists here. Using existing repository."
else
  echo "Initializing new git repository..."
  git init
fi

echo "Setting main as default branch..."
# Use main as default
if git rev-parse --verify main >/dev/null 2>&1; then
  git checkout main
else
  # If current branch is master, rename to main
  if git rev-parse --verify master >/dev/null 2>&1; then
    git branch -m master main
  else
    git checkout -b main
  fi
fi

echo "Configuring remote origin to: $REPO_URL"
if git remote get-url origin >/dev/null 2>&1; then
  git remote set-url origin "$REPO_URL"
else
  git remote add origin "$REPO_URL"
fi

# Basic gitignore fallback
if [ ! -f .gitignore ]; then
  echo "/dist/" > .gitignore
fi

# Ensure module tidy before commit
if command -v go >/dev/null 2>&1; then
  go mod tidy || true
fi

echo "Staging and committing changes..."
git add -A
if git diff --cached --quiet; then
  echo "No changes to commit."
else
  git commit -m "chore: initial import of Ollama Code"
fi

echo "Pushing to origin/main..."
git push -u origin main

echo "Done. Repo pushed: $REPO_URL"
