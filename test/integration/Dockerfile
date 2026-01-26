FROM golang:1.25-bookworm

RUN apt-get update && apt-get install -y \
    git \
    tmux \
    sqlite3 \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js (required for Claude CLI)
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install Claude CLI globally
RUN npm install -g @anthropic-ai/claude-code

# Create mock claude command for testing
# This mock accepts --settings flag and stays running to simulate real Claude
# Enables process inspection tests (view-agent-settings command)
RUN printf '#!/bin/bash\n\
# Mock claude command for integration tests\n\
# Parse --settings flag and stay running\n\
while [[ $# -gt 0 ]]; do\n\
  case $1 in\n\
    --settings)\n\
      SETTINGS="$2"\n\
      shift 2\n\
      ;;\n\
    *)\n\
      shift\n\
      ;;\n\
  esac\n\
done\n\
\n\
# Stay running like real Claude would\n\
sleep infinity\n' > /usr/local/bin/claude-mock \
    && chmod +x /usr/local/bin/claude-mock

# Override claude command with mock for tests
RUN mv /usr/local/bin/claude /usr/local/bin/claude-real 2>/dev/null || true \
    && ln -sf /usr/local/bin/claude-mock /usr/local/bin/claude

WORKDIR /app
