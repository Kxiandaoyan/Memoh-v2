# Web Search Skill Deployment Guide

## Problem Statement

When using the web-search skill in Cowork sessions, Claude needs to call bash scripts like:

```bash
bash SKILLs/web-search/scripts/search.sh "query" 10
```

However, this relative path approach has two critical issues:

1. **Development Mode**: Works fine when working directory is project root
2. **Production Mode**: After packaging, the `SKILLs` directory is bundled, and user's working directory is not the app installation directory

## Solution: Environment Variable Injection

### Overview

We inject a `SKILLS_ROOT` environment variable that points to the correct SKILLs directory in both development and production modes.

### Implementation

#### 1. Modified `electron/libs/coworkUtil.ts`

Added a `getSkillsRoot()` function that returns the correct path:

```typescript
function getSkillsRoot(): string {
  const appPath = app.getAppPath();

  // In development
  if (appPath.includes('node_modules') || !app.isPackaged) {
    const projectRoot = join(appPath, '../..');
    return join(projectRoot, 'SKILLs');
  }

  // In production, SKILLs are copied to userData
  return join(app.getPath('userData'), 'SKILLs');
}
```

And inject it into the environment:

```typescript
export async function getEnhancedEnv(): Promise<Record<string, string | undefined>> {
  // ... existing code ...

  // Inject SKILLs directory path for skill scripts
  const skillsRoot = getSkillsRoot();
  env.SKILLS_ROOT = skillsRoot;
  env.LOBSTERAI_SKILLS_ROOT = skillsRoot; // Alternative name for clarity

  // ... rest of code ...
}
```

#### 2. Updated `SKILLs/web-search/SKILL.md`

Changed all script invocations from:

```bash
bash SKILLs/web-search/scripts/search.sh "query" 10
```

To:

```bash
bash "$SKILLS_ROOT/web-search/scripts/search.sh" "query" 10
```

### How It Works

1. **Development Mode**:
   - `SKILLS_ROOT` → `/path/to/project/SKILLs`
   - Scripts execute from project directory
   - All relative paths work correctly

2. **Production Mode**:
   - `SKILLS_ROOT` → `~/Library/Application Support/lobsterai/SKILLs` (macOS)
   - Skills are copied to userData during first launch
   - Scripts execute from packaged location

3. **User Working Directory**:
   - Can be any directory user chooses
   - No longer needs to be project root
   - Skills always accessible via `$SKILLS_ROOT`

### Testing

```bash
# Set environment variable manually for testing
export SKILLS_ROOT="/xxx/SKILLs"

# Test search
bash "$SKILLS_ROOT/web-search/scripts/search.sh" "test query" 5
```

### Packaging Considerations

When packaging the application:

1. **Copy SKILLs to Resources**: Ensure `SKILLs` directory is copied to `extraResources` or `userData`
2. **Verify Paths**: Test that `getSkillsRoot()` returns correct path in packaged app
3. **Script Permissions**: Ensure bash scripts have execute permissions after copying

### Benefits

✅ **Cross-platform**: Works on macOS, Windows, Linux
✅ **Development-friendly**: No changes needed for local development
✅ **Production-ready**: Handles packaged app correctly
✅ **User-friendly**: Users can set any working directory
✅ **Maintainable**: Single source of truth for SKILLs location

### Migration Guide for Other Skills

If you create new skills that need to call scripts, always use:

```bash
bash "$SKILLS_ROOT/your-skill/scripts/your-script.sh"
```

Never use:

```bash
bash SKILLs/your-skill/scripts/your-script.sh  # ❌ Won't work in production
```

### Troubleshooting

**Problem**: `SKILLS_ROOT not found`

**Solution**: Ensure Electron app is recompiled after changes to `coworkUtil.ts`

```bash
npm run build
npm run electron:dev
```

**Problem**: Scripts not found in production

**Solution**: Verify SKILLs are included in packaging:

- Check `electron-builder.yml` or packaging config
- Ensure `extraResources` includes `SKILLs/**/*`
- Verify file permissions for bash scripts

**Problem**: Permission denied when running scripts

**Solution**: Set execute permissions:

```bash
chmod +x "$SKILLS_ROOT/web-search/scripts/*.sh"
```

### Related Files

- `electron/libs/coworkUtil.ts` - Environment variable injection
- `electron/libs/coworkRunner.ts` - Cowork execution engine
- `SKILLs/web-search/SKILL.md` - Updated skill documentation
- `electron/skillManager.ts` - Skill directory management
- `electron/skillServices.ts` - Background service management

### Future Improvements

1. **Auto-copy on First Launch**: Automatically copy bundled SKILLs to userData on first app launch
2. **Version Management**: Track skill versions and auto-update mechanism
3. **Skill Registry**: Central registry for installed skills with metadata
4. **Path Validation**: Add startup checks to verify `SKILLS_ROOT` is accessible

---

Last Updated: 2026-02-08
