# Git Commit and PR Guidelines

## No AI Tool References

NEVER include references to AI tools (Claude, ChatGPT, Copilot, etc.) in:
- Commit messages
- Commit trailers (Co-Authored-By, etc.)
- Pull request titles
- Pull request descriptions
- Code comments

All commits and PRs should appear as if written entirely by the human developer.

## Examples

❌ **WRONG:**
```
feat: add new feature

Implemented with help from Claude.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

✅ **CORRECT:**
```
feat: add new feature

Implements XYZ functionality with the following features:
- Feature A
- Feature B
```

## Commit Message Format

Follow conventional commits format:
```
<type>: <description>

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
