# Contributing to libredesk

Thanks for your interest in contributing.

## Before you start

- All contributions should align with the project's scope, goals, and technical direction.
- Bug fixes and documentation improvements can go straight to a PR; no prior discussion is needed.
- Anything larger should start with an issue. See below for why.
- Keep changes focused on a single concern wherever possible.
- For translations, please contribute through [Crowdin](https://crowdin.com/project/libredesk) instead of opening PRs for translation changes.

## Pull Requests

PRs are tricky. Any change, whether a new feature or a small tweak, has to fit the project's scope, goals, and technical direction. The code needs to match libredesk's existing style and conventions, and its impact on performance, usability, and stability should be clear before it can be merged.

That makes reviewing a slow, careful process. Larger PRs are harder to reason about, and reviewing them properly, discussing changes, and confirming they're safe to merge often takes more effort than writing the PR did. Whether a PR is accepted, and on what timeline, is ultimately up to the maintainers. This isn't personal. Maintainers carry the long-term context for the project, so decisions on direction and trade-offs sit with them.

To keep things moving:

1. **Propose first.** Open an issue describing what you want to build, why it fits libredesk, and how you plan to implement it, with relevant technical detail, before writing code. The change might not align with the project's direction, or it could already be in progress. Checking first saves everyone time.
2. **Keep PRs small.** Send focused PRs with a clear, single scope wherever possible. Smaller PRs are easier to review and test. Bundling multiple features into one PR is strongly discouraged.

Thanks for helping keep libredesk focused and maintainable.

## Backend translations (Gettext)

### How it works

The Go backend uses [leonelquinteros/gotext](https://github.com/leonelquinteros/gotext).
English source messages are the message IDs; there is no English catalog.
Unsupported languages and missing translations fall back to the English source.

There are two domains:

- `locales/<language>/LC_MESSAGES/backend.po`: messages used by the Go backend.
- `locales/<language>/LC_MESSAGES/templates.po`: messages used by
  server-rendered HTML templates and the Go functions that prepare their
  translated text.

### Writing translatable messages

Keep message IDs and domain names as string literals. Use Go printf arguments,
not named brace placeholders or concatenated sentences:

```go
i18n.Get("Could not load %s", name)
i18n.Get("%[1]s assigned the conversation to %[2]s", actor, assignee)
i18n.GetN("Attachment", "Attachments", len(attachments))
locale.GetND("templates", "%d result", "%d results", count, count)
```

The count selects the plural form; pass it again as a formatting argument when
the message contains `%d`. Explicit argument indexes allow translators to
reorder arguments. In Go HTML templates, always select the template domain:

```gotemplate
{{ .L.GetD "templates" "Search" }}
{{ .L.GetND "templates" "%d result" "%d results" .Total .Total }}
```

### Updating translation catalogs

From the repository root, run (Go required):

```sh
make i18n-extract   # Update the .pot files from source messages.
```

Commit only the updated `.pot` files. Translations must be edited through
[Crowdin](https://crowdin.com/project/libredesk), not directly in the
repository.

### Local development and validation

Compiled `.mo` files are generated locally and ignored by Git. Run
`make i18n-compile` before `make i18n-check`.

You can run these commands from the repository root:

- `make i18n-extract`: extract messages from Go code and HTML templates into
  the two `.pot` files. Requires Go only.
- `make i18n-update`: run extraction, merge the `.pot` files into the local
  `.po` files without fuzzy matching, and remove obsolete entries. Requires Go,
  `msgmerge` and `msgattrib`.
- `make i18n-compile`: validate the `.po` files and compile missing or outdated
  `.mo` files. Requires `msgfmt`.
- `make i18n-check`: verify that the `.pot` files match the source code, the
  `.po` files contain the expected messages, and each `.mo` file matches
  its corresponding `.po` file. Requires Go, `msgfmt` and `msgcmp`.

> [!NOTE]
> The `msgmerge`, `msgattrib`, `msgfmt` and `msgcmp` tools are provided by
> [GNU gettext](https://www.gnu.org/software/gettext/). Install the `gettext`
> package using your favorite package manager on Linux and macOS, or download
> pre-built binaries for Windows [here](
> https://mlocati.github.io/articles/gettext-iconv-windows.html).

`make build-backend` calls `i18n-compile`, then builds the Go backend with the
catalogs embedded in the binary. It requires Go, plus `msgfmt` when catalogs
need recompiling. The loader prefers `.mo` files and falls back to `.po` files
when the corresponding `.mo` file is absent. Recompile local `.po` changes and
rebuild the backend before testing them to avoid using an older embedded
translation.
