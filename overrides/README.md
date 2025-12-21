# MkDocs Material Theme Overrides

This directory contains customizations to the [MkDocs Material](https://squidfunk.github.io/mkdocs-material/) theme using the official [theme extension](https://squidfunk.github.io/mkdocs-material/customization/#extending-the-theme) mechanism.

## Files

### `main.html`

Overrides the `scripts` block to add TTL-based cache expiration for repository facts (stars, forks, version).

**Problem solved:** MkDocs Material caches GitHub repository data in `sessionStorage` indefinitely (until the browser tab is closed). This means users who visited the site before a release would see stale version numbers until they manually cleared their browser data.

**Solution:** A small script that:
1. Checks if cached repository facts exist with a timestamp
2. If older than 1 hour (TTL), clears the cache
3. Patches `__md_set` to record timestamps when new data is cached

**TTL Configuration:** Edit `TTL_MS` in `main.html` (default: 1 hour = 3600000ms)

## How Theme Extension Works

From [MkDocs Material Customization Docs](https://squidfunk.github.io/mkdocs-material/customization/):

1. **Setup:** Add `custom_dir: overrides` under `theme:` in `mkdocs.yml`
2. **Structure:** Mirror the theme's directory structure in `overrides/`
3. **Blocks:** Override template blocks by extending `base.html`

### Available Template Blocks

| Block name   | Purpose                                        |
|:-------------|:-----------------------------------------------|
| `scripts`    | Wraps the JavaScript application (footer)      |
| `extrahead`  | Empty block to add custom meta tags            |
| `styles`     | Wraps the style sheets                         |
| `header`     | Wraps the fixed header bar                     |
| `footer`     | Wraps the footer with navigation and copyright |
| `content`    | Wraps the main content                         |

Full list: https://squidfunk.github.io/mkdocs-material/customization/#overriding-blocks

### Block Override Pattern

```html
{% extends "base.html" %}

{% block scripts %}
  <!-- Scripts that run BEFORE Material's JS -->
  {{ super() }}
  <!-- Scripts that run AFTER Material's JS -->
{% endblock %}
```

## References

- [Theme Extension Guide](https://squidfunk.github.io/mkdocs-material/customization/#extending-the-theme)
- [Adding a Git Repository](https://squidfunk.github.io/mkdocs-material/setup/adding-a-git-repository/) - How repo stats are displayed
- [Source Code: components/source](https://github.com/squidfunk/mkdocs-material/tree/master/src/templates/assets/javascripts/components/source) - Where caching happens

## Storage Keys

MkDocs Material uses these storage keys (scoped by pathname):

| Key | Storage | Purpose |
|-----|---------|---------|
| `__source` | sessionStorage | Cached GitHub repo facts (stars, forks, version) |
| `__source_ts` | sessionStorage | TTL timestamp (added by our override) |
| `__palette` | localStorage | Color scheme preference |
| `__outdated` | sessionStorage | Version warning dismissed state |
