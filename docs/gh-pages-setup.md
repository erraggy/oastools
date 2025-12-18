# GitHub Pages Setup Guide

This guide outlines the steps required to configure your GitHub repository to host the `oastools` documentation using GitHub Pages.

## 1. Enable GitHub Pages

1.  Navigate to your repository on GitHub: [https://github.com/erraggy/oastools](https://github.com/erraggy/oastools)
2.  Click on the **Settings** tab.
3.  In the left sidebar, under the "Code and automation" section, click on **Pages**.
4.  Under **Build and deployment**:
    *   **Source**: Select **GitHub Actions**.

    *Note: Since we are using a custom workflow (`.github/workflows/docs.yml`) to build and deploy the site, selecting "GitHub Actions" tells GitHub to rely on our workflow rather than looking for a branch/folder to deploy directly.*

## 2. Verify Workflow Permissions

The deployment workflow requires permission to write to your repository's pages environment.

1.  In **Settings**, go to **Actions** -> **General** in the left sidebar.
2.  Scroll down to **Workflow permissions**.
3.  Ensure **Read and write permissions** is selected.
4.  Click **Save**.

*Alternatively, the `docs.yml` workflow file explicitly sets the necessary permissions:*
```yaml
permissions:
  contents: write
```
*So the default repository setting might be sufficient, but "Read and write" ensures no permission issues.*

## 3. Configure Environment Protection (Optional)

If you want to control deployments to the documentation site:

1.  In **Settings**, click on **Environments** in the left sidebar.
2.  You should see a `github-pages` environment (created automatically after the first successful deployment).
3.  You can configure protection rules here, such as requiring approval before deployment.

## 4. Trigger the First Deployment

The documentation workflow is configured to run on pushes to the `main` branch.

1.  Push the changes from this branch (`feature/gh-pages-setup`) to `main` (via a Pull Request).
2.  Once merged, go to the **Actions** tab in your repository.
3.  You should see the **Publish Docs** workflow running.
4.  Once completed, your site will be live at: [https://erraggy.github.io/oastools/](https://erraggy.github.io/oastools/)

## 5. Verify the Site

After the workflow finishes:

1.  Visit [https://erraggy.github.io/oastools/](https://erraggy.github.io/oastools/).
2.  Check that the navigation links work correctly.
3.  Verify that the "Deep Dive" package documentation pages load.

## Troubleshooting

*   **404 Not Found**: Ensure the `site_url` in `mkdocs.yml` matches your actual GitHub Pages URL. Currently it is set to `https://erraggy.github.io/oastools/`.
*   **Workflow Failure**: Check the logs in the **Actions** tab for specific error messages.
*   **Missing Content**: Ensure `scripts/prepare-docs.sh` is copying all necessary files.
