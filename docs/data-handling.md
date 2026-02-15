# Data Handling Reference

Concise reference for anyone editing capsule pipeline scripts or Go subprocess code. These principles prevent silent data corruption when processing untrusted content (task descriptions, LLM output, user feedback).

## 1. JSON Extraction

Always use `jq -r '.field // empty'` for optional fields. Use `-e` for required fields (exits non-zero on null/false).

```bash
# Good: null-safe optional field
TITLE=$(echo "$JSON" | jq -r '.[0].title // empty')

# Good: required field — fails if missing
TITLE=$(echo "$JSON" | jq -e -r '.[0].title') || { echo "missing title" >&2; exit 1; }

# Bad: grep/sed on JSON
TITLE=$(echo "$JSON" | grep '"title"' | sed 's/.*: "\(.*\)"/\1/')
```

Validate types before use. An array field might contain unexpected element types.

## 2. Template Rendering

Template rendering uses POSIX `awk` with `-v` variable assignments, which avoids shell expansion. The awk renderer handles Go `text/template` syntax: `{{.Field}}` substitution and `{{if .Field}}...{{end}}` conditionals.

```bash
# Safe: awk -v passes values without shell expansion
awk -v TaskID="$TASK_ID" -v Title="$TITLE" '...' template.md > output.md
```

**Caveat:** `awk -v` interprets C-style escape sequences (`\n`, `\t`, `\\`). If bead content contains literal backslash sequences, they will be interpreted. For most bead descriptions this is not an issue.

The previous `envsubst` approach was replaced because it had a self-referencing risk: if a variable's value contained `${LISTED_VAR}`, envsubst would expand it recursively.

## 3. Bash String Replacement

`${var//pattern/replacement}` treats `&` as a backreference to the matched text and `\` as an escape character in the replacement string. **Never use this for untrusted content.**

```bash
# Dangerous: if CONTENT contains & or \, output is corrupted
RESULT="${TEMPLATE//\{\{PLACEHOLDER\}\}/$CONTENT}"

# Safe: awk with file-based substitution
printf '%s\n' "$CONTENT" > "$tmpfile"
RESULT=$(awk -v f="$tmpfile" '/\{\{PLACEHOLDER\}\}/{while((getline l<f)>0)print l;close(f);next}{print}' "$TEMPLATE_FILE")
```

Note: `awk gsub()` also treats `&` as backreference in the replacement. For multiline untrusted content, use the file-read approach above.

## 4. Variable Quoting

Always double-quote: `"$var"`. Double-quoted expansion does **not** recursively evaluate `$(...)` or backticks within the value — it performs simple parameter substitution. This is safe.

```bash
# Safe: $feedback may contain $(rm -rf /) but it's treated as literal text
PROMPT="Review feedback: $feedback"

# DANGEROUS: eval re-parses the string, executing embedded commands
eval "$var"   # Never use eval with untrusted data
```

## 5. printf vs echo

Always use `printf '%s\n' "$var"` for arbitrary content. `echo` interprets `-n`, `-e` flags and may interpret backslash escapes depending on shell configuration.

```bash
# Good: handles any content safely
printf '%s\n' "$arbitrary_content" > output.txt

# Bad: breaks if content starts with -n, -e, or contains \n
echo "$arbitrary_content" > output.txt
```

## 6. Passing Data Between Scripts

| Method | Safe for | Limits |
|--------|----------|--------|
| CLI args with `"$var"` | Moderate strings | ARG_MAX (~2MB on Linux) |
| stdin / temp files | Large/multiline data | Disk space |
| Environment variables | Moderate strings | ~128KB combined on Linux |

Prefer stdin or temp files for large or multiline data. Always clean up temp files in a trap.

## 7. Signal Parsing

Validate all required fields **and** their types. Array elements should be type-checked, not just the array itself.

```bash
# Incomplete: only checks that files_changed is an array
jq '.files_changed | type == "array"'

# Complete: checks array AND that all elements are strings
jq '.files_changed | type == "array" and all(type == "string")'
```

## 8. Boundary Principle

Sanitize data at every format crossing:

- **JSON → shell**: Use `jq -r` with `// empty`, never grep/sed
- **Shell → template**: Use `awk -v` for variable passing, never raw `${//}`
- **Shell → markdown**: Content is generally safe (markdown doesn't execute), but beware of template placeholders in content
- **Shell → git**: Quote arguments, use `--` to separate flags from paths
- **Go → subprocess**: Use `exec.Command` with separate arguments, never `"sh", "-c", concatenated`

## 9. Go Subprocess Safety

The same boundary principles apply when Go code shells out to git or other tools via `exec.Command`. See also `docs/go-conventions.md` section 10.

### Argument Separation

`exec.Command` passes arguments directly to the process (no shell interpretation). This is inherently safe from command injection — but not from flag injection:

```go
// Safe from command injection (no shell)
cmd := exec.Command("git", "branch", "-D", branchName)

// UNSAFE: shell interprets special chars, semicolons, pipes
cmd := exec.Command("sh", "-c", "git branch -D "+branchName)
```

### Flag Injection

Even with argument separation, an ID like `--version` or `-rf` can be interpreted as a flag by the target command. Validate inputs at the boundary:

```go
// Reject values that look like flags
if strings.HasPrefix(id, "-") {
    return fmt.Errorf("invalid id: must not start with -")
}
```

For git commands accepting paths, use `--` to signal end-of-flags:

```bash
# Shell: -- prevents $FILE from being interpreted as a flag
git checkout -- "$FILE"
```

### Git Worktree Operations

Special considerations for `git worktree` commands:

- **`git worktree remove --force`** discards uncommitted changes silently. Document this in godoc if used.
- **`git worktree prune`** cleans orphaned metadata after manual directory deletion. Call after bulk removal operations.
- **Always use `git worktree remove`** instead of `rm -rf` — manual deletion orphans git's internal tracking in `.git/worktrees/`.

## Existing Correct Patterns

These patterns in the codebase are correct — reference them:

- `printf '%s\n'` for writing arbitrary content (`run-phase.sh`)
- `jq -r '... // empty'` for null-safe field extraction (throughout)
- `awk -v Var="$SHELL_VAR"` for template rendering (`prep.sh`)
- `exec.Command("git", arg1, arg2)` with separate arguments (`worktree.go`, `claude.go`)
- `validateID()` rejecting `-`, `/\`, `.`, `..` before path construction (`worktree.go`)
