# GitHub Pages Troubleshooting

## Quick Fix for 404 Error

The 404 error is likely because:

1. **The workflow hasn't run yet** - It needs to be triggered
2. **GitHub Pages needs to be configured** - Set source to "GitHub Actions"

## Immediate Actions

### 1. Verify GitHub Pages Settings

Go to: `https://github.com/bondowe/webfram/settings/pages`

Ensure:
- **Source** is set to **"GitHub Actions"** (not "Deploy from a branch")

### 2. Trigger the Workflow

**Option A: Push the new changes**
```bash
git add .
git commit -m "Fix GitHub Pages deployment"
git push
```

The workflow will now trigger on pushes to `main` that affect `docs/`

**Option B: Manual trigger**
1. Go to: `https://github.com/bondowe/webfram/actions`
2. Click "Deploy Documentation to GitHub Pages"
3. Click "Run workflow"
4. Select `main` branch
5. Click "Run workflow" button

### 3. Wait for Deployment

- Check workflow progress at: `https://github.com/bondowe/webfram/actions`
- Wait 2-5 minutes for completion
- Your site will be at: `https://bondowe.github.io/webfram/`

## What Changed

The workflow has been updated to:
1. ✅ **Build Jekyll properly** instead of just copying files
2. ✅ **Trigger on pushes to docs/** for easier testing
3. ✅ **Include Gemfile** with all required dependencies

## Testing Locally

To test the site locally before deployment:

```bash
cd docs

# Install dependencies (first time only)
bundle install

# Run Jekyll locally
bundle exec jekyll serve

# Visit http://localhost:4000/webfram/
```

## Common Issues

### "Source: GitHub Actions" not available

**Solution**: 
1. Go to Settings → Actions → General
2. Under "Workflow permissions", select "Read and write permissions"
3. Check "Allow GitHub Actions to create and approve pull requests"
4. Save changes
5. Return to Settings → Pages
6. "GitHub Actions" should now be available as a source

### Workflow fails with Ruby/Jekyll errors

**Solution**: The new Gemfile should fix this. If issues persist:
1. Check the Actions log for specific errors
2. Verify all files are committed and pushed

### Site shows old content

**Solution**:
1. Clear browser cache
2. Try incognito/private browsing mode
3. Wait a few minutes for CDN to update

### Wrong baseurl

If pages load but styling is broken:
1. Check `docs/_config.yml` has `baseurl: /webfram`
2. Verify repository name is exactly "webfram"

## Quick Verification Commands

```bash
# Verify files are committed
git status

# Verify files are pushed
git log origin/main..HEAD

# If commits exist locally, push them
git push

# Check if workflow file is valid
cat .github/workflows/deploy-docs.yml
```

## Next Steps

1. **Commit and push** the new Gemfile and updated workflow
2. **Check Actions tab** to see if workflow is running
3. **Wait for green checkmark** in Actions
4. **Visit** `https://bondowe.github.io/webfram/`
5. **Success!** 🎉

## Still Having Issues?

Check:
- [ ] Repository is public (or you have GitHub Pro for private repo Pages)
- [ ] Workflow has run successfully (green checkmark in Actions)
- [ ] GitHub Pages source is set to "GitHub Actions"
- [ ] URL is exactly `https://bondowe.github.io/webfram/` (with trailing slash)
- [ ] Browser cache is cleared
