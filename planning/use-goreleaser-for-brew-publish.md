## Full Steps to Automate with GoReleaser
1. Set up a Homebrew Tap Repository:
   Create a new public GitHub repository specifically for your Homebrew tap.
   This repository's name must be prefixed with `homebrew-` (so: `erraggy/homebrew-oastools`). This allows users to tap it with `brew tap erraggy/oastools`.
2. Configure GoReleaser:
   Install GoReleaser: `brew install goreleaser` (or follow other installation methods).
   Initialize GoReleaser in your Go CLI project: `goreleaser init`. This generates a `.goreleaser.yaml` file.
   Edit `.goreleaser.yaml` to include a brews section for Homebrew:
   ```yaml
   brews:
    - name: oastools # The name of your CLI in Homebrew
      repository:
      owner: erraggy # Your GitHub username
      name: homebrew-oastools # The name of your Homebrew tap repository
      commit_author:
      name: Robbie Coleman
      email: robbie@robnrob.com
      homepage: "https://github.com/erraggy/oastools" # Optional: your project homepage
      description: "OpenAPI Specification (OAS) tools for validation, parsing, converting, and joining specs." # Optional: a description
      license: "MIT" # Optional: your project's license
   ```
3. Set up GitHub Token for GoReleaser:
   GoReleaser needs a GitHub Personal Access Token (PAT) with `repo` scope to publish releases and update the Homebrew tap.
   Create a PAT in your GitHub settings.
   Set this PAT as an environment variable `GITHUB_TOKEN` in your CI/CD pipeline or local environment when running GoReleaser.
4. Create a Release Tag:
   GoReleaser uses Git tags to determine release versions.
   Create an annotated Git tag for your release (e.g., `git tag -a v1.0.0 -m "Release v1.0.0"`).
   Push the tag to your GitHub repository: `git push origin v1.0.0`.
5. Run GoReleaser:
   Execute `goreleaser release --rm-dist` in your CLI project's root directory.
   This command will:
   - Build and package your Go CLI for various platforms.
   - Create a new GitHub release with the generated artifacts.
   - Generate the Homebrew formula (a Ruby file) and commit it to your `homebrew-oastools` tap repository.
6. User Installation:
   - Users can then install your CLI via Homebrew by first tapping your repository and then installing the formula:
     ```shell
     brew tap erraggy/oastools
     brew install oastools
     ```
