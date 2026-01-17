# Rocha Project TODO List

## rocha setup - Dependency Checks

- [ ] Implement tmux installation check in setup command
- [ ] Implement git installation check in setup command
- [ ] Add helpful error messages for missing dependencies

## rocha add-session - Session Creation

- [ ] Create add-session command
- [ ] Implement current directory detection in add-session
- [ ] Implement git branch detection in add-session
- [ ] Add session state persistence in add-session
- [ ] Validate branch names (git-compliant naming)

## attach - Session Attachment

- [ ] Create attach command
- [ ] Implement session existence check in attach
- [ ] Implement tmux session running check in attach
- [ ] Implement tmux session start/attach logic in attach

## Auto-refresh Feature

- [ ] Implement auto-update of session list when session status changes

## UI Improvements

- [ ] Allow mouse selection on the session list
- [ ] Implement copy/paste functionality
- [ ] Allow filtering session list with sticky filters
- [ ] Configure tmux to allow mouse scroll

## Configuration and State Management

- [ ] Move state.json to ~/.rocha directory
- [ ] Auto-enable debug mode when --debug-file is set
- [ ] Add settings.json to rocha so we can configure stuff permanently

## Code Organization

- [ ] Move packages to the internal folder

## Distribution

- [ ] Add Homebrew support

## Bug Fixes

- [x] Fix worktree removal failing when directory has modified/untracked files
- [ ] Fix subprocess logging - hook commands not logging to shared log file
