# GitHub Pages Setup

This guide explains how to enable GitHub Pages for WebFram documentation.

## Configuration Created

The following files have been created to enable professional documentation on GitHub Pages:

### 1. GitHub Actions Workflow
- **File**: `.github/workflows/deploy-docs.yml`
- **Triggers**: On release publication or manual dispatch
- **Action**: Deploys `docs/` folder to GitHub Pages

### 2. Jekyll Configuration
- **File**: `docs/_config.yml`
- **Theme**: [just-the-docs](https://just-the-docs.github.io/just-the-docs/) - A modern, professional documentation theme
- **Features**:
  - Search functionality
  - Copy code button
  - Responsive design
  - Custom color scheme
  - Navigation structure
  - Footer with copyright

### 3. Custom Color Scheme
- **File**: `docs/_sass/color_schemes/webfram.scss`
- **Styling**: Custom WebFram-branded colors

## Enable GitHub Pages

Follow these steps to enable GitHub Pages for your repository:

### Step 1: Repository Settings

1. Go to your repository on GitHub: `https://github.com/bondowe/webfram`
2. Click on **Settings** (gear icon)
3. Scroll down to **Pages** in the left sidebar

### Step 2: Configure GitHub Pages

1. Under **Source**, select:
   - **Source**: GitHub Actions
2. Click **Save**

### Step 3: Enable Workflow Permissions

1. In **Settings**, go to **Actions** → **General**
2. Scroll to **Workflow permissions**
3. Select **Read and write permissions**
4. Check **Allow GitHub Actions to create and approve pull requests**
5. Click **Save**

### Step 4: Trigger Deployment

You have two options to deploy:

**Option A: Create a Release**
1. Go to **Releases** in your repository
2. Click **Draft a new release**
3. Create a tag (e.g., `v1.0.0`)
4. Publish the release
5. The workflow will automatically deploy the docs

**Option B: Manual Trigger**
1. Go to **Actions** tab
2. Select **Deploy Documentation to GitHub Pages** workflow
3. Click **Run workflow**
4. Select the `main` branch
5. Click **Run workflow**

### Step 5: Access Your Documentation

After deployment completes (2-5 minutes), your documentation will be available at:

```
https://bondowe.github.io/webfram/
```

## Documentation Structure

The deployed site will have:

- **Home**: Landing page with navigation (`index.md`)
- **Search**: Full-text search across all documentation
- **Navigation**: Organized sidebar with all documentation pages
- **Responsive**: Mobile-friendly design
- **Professional**: Clean, modern theme

## Local Preview

To preview the documentation locally with Jekyll:

### Install Jekyll

```bash
# On macOS/Linux
gem install bundler jekyll

# On Windows
# Follow: https://jekyllrb.com/docs/installation/windows/
```

### Create Gemfile

Create `docs/Gemfile`:

```ruby
source 'https://rubygems.org'

gem 'jekyll', '~> 4.3'
gem 'just-the-docs'
gem 'jekyll-seo-tag'
gem 'jekyll-github-metadata'
gem 'jekyll-include-cache'
```

### Run Locally

```bash
cd docs
bundle install
bundle exec jekyll serve
```

Visit `http://localhost:4000/webfram/`

## Theme Customization

The Just the Docs theme is highly customizable. Edit `docs/_config.yml` to:

- Change colors (or edit `docs/_sass/color_schemes/webfram.scss`)
- Modify navigation structure
- Add custom CSS/JavaScript
- Configure search options
- Add Google Analytics

See [Just the Docs documentation](https://just-the-docs.github.io/just-the-docs/) for all options.

## Troubleshooting

### Workflow Fails

- Check **Actions** tab for error details
- Ensure workflow permissions are set correctly
- Verify the `docs/` folder exists

### 404 Error

- Wait 2-5 minutes after deployment
- Clear browser cache
- Check GitHub Pages settings are correct

### Styling Issues

- Verify `_config.yml` is in the `docs/` folder
- Check that baseurl matches your repository name
- Ensure custom SCSS file is in correct location

## Next Steps

1. **Enable GitHub Pages** following steps above
2. **Create your first release** to trigger automatic deployment
3. **Customize** the theme in `docs/_config.yml` as needed
4. **Add** a custom domain (optional) in repository settings

## Support

- [Just the Docs Documentation](https://just-the-docs.github.io/just-the-docs/)
- [GitHub Pages Documentation](https://docs.github.com/en/pages)
- [Jekyll Documentation](https://jekyllrb.com/docs/)
