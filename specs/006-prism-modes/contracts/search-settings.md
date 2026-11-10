# Contract: settings

## Module `prism-search` (`modules.prism-search.*`)

All are override-only, validated by the existing module rules, applied to the next search (read from `api.settings.values` at each query), and shown by `lumen modules show prism-search`.

| Key | Type | Default | Range / rule | Meaning |
|-----|------|---------|--------------|---------|
| `rowsPerGroup` | integer | 20 | 1 to 200 | Rows shown per file group, apps list and commands list; the rest becomes the "N more" row |
| `maxFiles` | integer | 200000 | 1000 to 5000000 | Files looked at per stage before it stops (`cut`) |
| `searchTimeMs` | integer | 1500 | 100 to 30000 | Time budget per stage (also stops slow mounts) |
| `hiddenFiles` | boolean | false | | Match hidden files and folders inside searched folders |
| `hiddenHomeDirs` | boolean | false | | Also search hidden folders of `~` other than `.cache`, `.local`, `.config` |
| `systemGroup` | boolean | true | | Show the collapsed system files group |
| `systemRoots` | stringlist | `["/etc","/usr","/opt","/var","/srv"]` | up to 16 absolute paths | Where system files are searched; `/proc`, `/sys`, `/dev`, `/run`, `/tmp` are never walked |
| `commandDirs` | stringlist | `[]` | up to 16 absolute paths | Extra command folders after the default ones |
| `grepMaxMatches` | integer | 500 | 10 to 10000 | Matches before grep stops |
| `grepCase` | string | `smart` | `smart`, `ignore`, `match` | Case rule for grep |
| `searchUrl` | string | `https://duckduckgo.com/?q=%s` | `http(s)` URL with exactly one `%s`, at most 200 chars | The web search engine |
| `terminal` | string | `""` | max 128 chars | Terminal program for the editor and for `Terminal=true` apps; empty means `TERMINAL` |
| `terminalExecFlag` | string | null (unset) | max 16 chars | Flag that precedes the command in the terminal; unset follows the terminal's known flag, else `-e` |

A bad value falls back for that key with an error naming `modules.prism-search` and the key. A relative path in `systemRoots` or `commandDirs` is a bad entry (dropped and reported, as for any stringlist); a missing directory is skipped with one warning in `lumen modules show`.

The wording of the hints and messages of the source is fixed text of the module (it is not a visible component, only strings shown in Prism's message line, which Prism styles).

## Prism (`modules.prism.*`), added in Prism 1.1.0

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `matchRole` | string (theme role) | `primary` | Color of marked (matching) text in a row |
| `messageRole` | string (theme role) | `onSurfaceVariant` | Color of the message line (`info`) |
| `errorRole` | string (theme role) | `error` | Color of the message line (`error`) |

Role lists are the same 16 roles as the other color settings; the existing test that keeps the list equal to `Theme.qml` covers them. The font of the message line is the empty-state font (no new font keys).
