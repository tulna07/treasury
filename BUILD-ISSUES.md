# Docker Build Issues — Root Cause & Fix Report

## Issue 1 — `migrate/migrate:v4` image tag does not exist

**File:** `.github/workflows/build.yml`

**Problem:** The Docker Hub image `migrate/migrate:v4` does not exist. The tag format used by the golang-migrate project is versioned releases (e.g. `v4.18.1`) or `latest`.

**Fix:** Changed to `migrate/migrate:latest` in both `Dockerfile.migrate` and `K8S-DEPLOY.md`.

---

## Issue 2 — `tsconfig.tsbuildinfo` committed to repo

**File:** `app/apps/web/tsconfig.tsbuildinfo`

**Problem:** This is a stale TypeScript incremental build cache file. When present in the repo, it causes Next.js builds to fail on clean CI environments because the cached state doesn't match the clean build context.

**Fix:** Removed from repo, added `tsconfig.tsbuildinfo` to `app/.gitignore`.

---

## Issue 3 — Turbopack cannot resolve CSS `@import` for pnpm virtual store packages

**File:** `app/apps/web/src/app/globals.css`, `app/apps/web/next.config.ts`

**Problem:** This was the main blocker. `globals.css` imports two CSS-only packages as bare specifiers:

```css
@import "tw-animate-css";
@import "shadcn/tailwind.css";
```

Next.js 16 uses Turbopack as the default bundler for production builds. Turbopack's CSS `@import` resolver cannot follow pnpm's virtual store symlinks (`node_modules/.pnpm/`) to find these packages. This fails regardless of hoisting strategy, Docker stage structure, or environment variables — it is a Turbopack limitation with pnpm monorepos.

**What was tried and failed:**
| Attempt | Why it failed |
|---|---|
| `pnpm install --shamefully-hoist` | Hoists JS packages but Turbopack CSS resolver still can't find them |
| `NEXT_TURBOPACK=0` env var | Next.js 16 ignores this variable |
| `TURBOPACK=0` in npm build script | Next.js 16 ignores this variable |
| `next build --no-turbopack` | Flag does not exist in Next.js 16 |
| Copying `node_modules` between Docker stages | Breaks pnpm's relative symlinks |
| `turbopack.resolveAlias` with `require.resolve()` | `tw-animate-css` is CSS-only with no JS exports — `require.resolve` throws `ERR_PACKAGE_PATH_NOT_EXPORTED` and crashes `next.config.ts` |
| `turbopack.resolveAlias` with `"tw-animate-css/dist/tw-animate.css"` | Turbopack splits this into module `tw-animate-css` + subpath `/dist/tw-animate.css`, still fails module resolution |

**Fix:** Use `turbopack.resolveAlias` in `next.config.ts` with `./node_modules/...` relative paths — this tells Turbopack the exact file location, bypassing module resolution entirely:

```ts
turbopack: {
  resolveAlias: {
    "tw-animate-css": "./node_modules/tw-animate-css/dist/tw-animate.css",
    "shadcn/tailwind.css": "./node_modules/shadcn/dist/tailwind.css",
  },
},
```

The exact internal file paths were verified from the published npm tarballs:
- `tw-animate-css@1.4.0` → `dist/tw-animate.css`
- `shadcn@4.1.2` → `dist/tailwind.css`

---

## Summary

| # | Problem | File changed | Fix |
|---|---|---|---|
| 1 | `migrate/migrate:v4` tag doesn't exist | `Dockerfile.migrate`, `K8S-DEPLOY.md` | Use `migrate/migrate:latest` |
| 2 | Stale `tsconfig.tsbuildinfo` in repo | `app/.gitignore` | Remove file, add to gitignore |
| 3 | Turbopack CSS `@import` fails for pnpm virtual store packages | `app/apps/web/next.config.ts` | `turbopack.resolveAlias` with `./node_modules/` relative paths |
