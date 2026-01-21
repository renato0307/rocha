# Go Development Guidelines

## Style

When doing code make sure you:
- Don't use `interface{}`, use `any`
- Sort the struct fields alphabetically
- Split the imports into three groups: stdlib, dependencies, internal imports


## Development Pace

⏱️ **No urgency.** Take your time. Thoughtful design > speed.

## Core Principles (Quick Reference)

### 1. Single Responsibility Principle (SRP)
**Rule:** One struct/function = one reason to change

❌ `UserService` that validates + saves + emails + logs
✅ `UserValidator`, `UserRepository`, `EmailSender`, `Logger`

**Detection:** If you use "and" to describe it → violates SRP

### 2. DRY (Don't Repeat Yourself)
**Rule:** Duplicate code ≥ 3 times → extract it (rule of three)

❌ Same validation in `CreateUser()` and `UpdateUser()`
✅ Shared `validateUser()` called by both

**Exception:** Go Proverb #3 - copying < 10 lines is OK if it avoids dependency

### 3. Open/Closed Principle
**Rule:** Extend by adding code, not modifying existing code

❌ Adding methods to interface → breaks all implementations
✅ New interface implementations, composition, strategies

**Pattern:** Use small interfaces that compose together

## Go Proverbs (Quick Reference)

### 1. Small Interfaces (≤ 3 methods ideal)
❌ `interface { Method1(); Method2(); Method3(); Method4(); Method5() }`
✅ `interface { DoThing() error }` + composition

### 2. No interface{}/any (unless truly necessary)
❌ `func Process(data interface{})`
✅ `func ProcessUser(user *User)`
✅ `func ProcessOrder(order *Order)`

### 3. Copy > Dependency (for small code)
**Rule:** < 10 lines + used ≤ 2 times = copy it
❌ Import 50MB package for 5-line function
✅ Copy the 5-line function

### 4. Build Tags for Syscalls
**Pattern:** `//go:build unix` or `//go:build windows`
❌ `syscall.Umask()` in cross-platform file
✅ Separate `file_unix.go` and `file_windows.go`

### 5. Clear > Clever
❌ Dense one-liners, complex conditionals, "smart" tricks
✅ Named variables, explicit steps, readable flow

### 6. Errors as Values
❌ `func Do() Result` (what if it fails?)
✅ `func Do() (Result, error)`
✅ Custom error types when context needed

## Workflow

### Phase 1: Planning → Include Checklist

```markdown
## Design Principles Checklist

### SRP
- [ ] Each struct = one responsibility
- [ ] Each function = one thing at one abstraction level
- [ ] No "and" in descriptions

### DRY
- [ ] No duplicate logic (rule of three: abstract after 3+ uses)
- [ ] Shared utilities for common patterns

### Open/Closed
- [ ] Small, composable interfaces
- [ ] Extension via new code, not modification

### Go Proverbs
- [ ] Interfaces ≤ 3 methods
- [ ] No `interface{}`/`any` without justification
- [ ] Clear over clever
- [ ] All errors returned as values
- [ ] Small code copied, not imported
- [ ] Syscalls use build tags

### Logging & Errors
- [ ] Always use rocha logging package
- [ ] DO NOT use slog directly
- [ ] Add logs to ensure that we can use --debug to trace execution flow and troubleshoot issues
```

### Phase 2: Implementation → Write Code

(No special instructions - just code following the principles above)

### Phase 3: Self-Evaluation → Report Results

```markdown
## Code Quality Self-Evaluation

### SRP: ✅ / ⚠️ / ❌
- Component1: Responsibility
- Component2: Responsibility

### DRY: ✅ / ⚠️ / ❌
- Duplicates: [none/list them]
- Abstractions: [appropriate/premature]

### Open/Closed: ✅ / ⚠️ / ❌
- Extension points: [list]

### Go Proverbs: ✅ / ⚠️ / ❌
- Interfaces: [Name: N methods, Name: N methods]
- interface{}/any: [none/justified uses]
- Clarity: [clear/needs simplification]

### ⚠️ Violations
[List any violations with location and suggested fix, ask for guidance]
```

## Exception Reporting Template

When violations are found, report using:

```
⚠️ **Violation Detected**

**Principle:** [SRP/DRY/Open-Closed/Proverb]
**Location:** `package.Type.Method()`
**Issue:** [Brief description]

**Suggested fix:**
- Option 1: [description]
- Option 2: [description]

**Question:** [Ask user for guidance on how to proceed]
```

## Quick Decision Tree

```
Writing new code?
├─ Does more than one thing? → Violates SRP, split it
├─ Seen this logic 3+ times? → Violates DRY, extract it
├─ Interface > 3 methods? → Violates Proverb #1, split it
├─ Using interface{}/any? → Justify it or use concrete types
├─ Using syscall package? → Use build tags
└─ Code unclear/clever? → Simplify it
```

---

**Key:** These are guidelines, not absolute rules. Pragmatism over dogmatism. When in doubt, ask the user.
