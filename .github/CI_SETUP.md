# GitHub Actions CI/CD Setup Documentation

## Overview

This repository uses GitHub Actions for Continuous Integration and Continuous Deployment. The CI pipeline automatically runs on every push, pull request, and can be manually triggered.

## Workflow Configuration

The main CI workflow (`.github/workflows/ci.yml`) includes the following jobs:

### 1. **Test Job**

- **Runs on**: Ubuntu Latest
- **Go Versions**: 1.22, 1.23
- **Steps**:
  - Checkout code
  - Set up Go environment
  - Download and verify dependencies
  - Run tests with race detection and coverage
  - Merge coverage reports from multiple modules
  - Generate and display coverage report in workflow summary
  - Upload coverage to Codecov
  - Verify coverage meets 70% threshold
  - Upload coverage artifacts

### 2. **Build Job**

- **Runs on**: Ubuntu Latest
- **Go Versions**: 1.22, 1.23
- **Dependencies**: Requires test job to pass
- **Steps**:
  - Checkout code
  - Set up Go environment
  - Build all packages
  - Build example applications

### 3. **Lint Job**

- **Runs on**: Ubuntu Latest
- **Go Version**: 1.23
- **Steps**:
  - Checkout code
  - Set up Go environment
  - Run golangci-lint with comprehensive checks

### 4. **Create Issue on Failure**

- **Runs on**: Ubuntu Latest
- **Triggers**: Only on failure of previous jobs
- **Conditions**: Push, schedule, or manual dispatch events
- **Behavior**:
  - Creates a new GitHub issue when CI fails
  - If an issue already exists for the branch, adds a comment
  - Includes workflow run details, commit info, and failure summary
  - Labels: `ci-failure`, `bug`, `automated`

## Badges

The following badges are displayed in the README:

1. **CI Status**: Shows current build status
2. **Codecov**: Shows code coverage percentage
3. **Go Report Card**: Shows code quality grade
4. **Go Reference**: Links to pkg.go.dev documentation
5. **License**: Shows MIT license

## Codecov Integration

Coverage reports are uploaded to Codecov with the following configuration:

- **Target Coverage**: 70%
- **Precision**: 2 decimal places
- **Status Checks**: Enabled for project and patch
- **Comments**: Automatically added to PRs with coverage diff

### Setting up Codecov Token

To enable Codecov uploads:

1. Sign up at [codecov.io](https://codecov.io)
2. Add your repository
3. Copy the repository upload token
4. Add it as a secret named `CODECOV_TOKEN` in your GitHub repository settings

**Note**: For public repositories, Codecov token is optional but recommended.

## Required GitHub Secrets

| Secret Name | Description | Required |
|------------|-------------|----------|
| `CODECOV_TOKEN` | Codecov upload token | Optional for public repos |
| `GITHUB_TOKEN` | Automatically provided by GitHub | Auto-generated |

## Manual Workflow Triggers

You can manually trigger the CI workflow:

1. Go to the **Actions** tab
2. Select the **CI** workflow
3. Click **Run workflow**
4. Select the branch
5. Click **Run workflow** button

## Coverage Reports

Coverage reports are available in multiple formats:

1. **Workflow Summary**: View detailed coverage in the Actions run summary
2. **Codecov Dashboard**: Visit [codecov.io/gh/bondowe/webfram](https://codecov.io/gh/bondowe/webfram)
3. **Artifacts**: Download `coverage-reports` artifact from workflow runs (retained for 30 days)

## Automated Issue Creation

When the CI pipeline fails on protected branches (main, develop), the system:

1. Checks for existing open issues with the `ci-failure` label
2. If found, adds a comment with new failure details
3. If not found, creates a new issue with:
   - Branch name and commit SHA
   - Triggering actor
   - Link to workflow run
   - Commit message
   - Checklist for resolution

### Example Issue

```markdown
## CI Workflow Failed

**Branch:** main
**Commit:** abc1234
**Triggered by:** username
**Workflow Run:** [View Details](https://github.com/...)

### Commit Message
```

Add new feature

```

### Failed Jobs
One or more jobs in the CI workflow have failed. Please review the workflow run details above.

### Action Items
- [ ] Review the failed job logs
- [ ] Fix the underlying issue
- [ ] Re-run the workflow or push a fix
```

## Dependabot

Dependabot is configured to automatically create pull requests for:

- Go module dependencies (weekly on Monday)
- OpenAPI submodule dependencies (weekly on Monday)
- GitHub Actions versions (weekly on Monday)

## Troubleshooting

### Tests Failing Locally but Passing in CI

- Ensure you're using the correct Go version
- Run `go mod download && go mod verify`
- Check for race conditions using `go test -race ./...`

### Coverage Below Threshold

- Run locally: `go test -coverprofile=coverage.out ./...`
- View coverage: `go tool cover -html=coverage.out`
- Add tests for uncovered code paths

### Lint Failures

- Run locally: `golangci-lint run`
- Fix issues automatically: `golangci-lint run --fix`
- Check `.golangci.yml` for configured rules

### Workflow Not Triggering

- Check branch protection rules
- Verify workflow file syntax
- Check workflow permissions in repository settings

## Best Practices

1. **Always run tests locally** before pushing
2. **Keep coverage above 70%** for all changes
3. **Address lint issues** before creating PR
4. **Write descriptive commit messages** for better issue tracking
5. **Review coverage reports** in PR comments
6. **Monitor Dependabot PRs** and merge regularly

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Codecov Documentation](https://docs.codecov.io)
- [golangci-lint Documentation](https://golangci-lint.run)
- [Go Testing Documentation](https://golang.org/pkg/testing/)

---

*Last Updated: November 6, 2025*
