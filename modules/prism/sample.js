.pragma library

// The fixed sample rows of Prism's preview mode (specs/005-prism-window, contracts/
// prism-surface.md "Preview mode"). They are not search results and do nothing when chosen.
// Pure functions, tested in scripts/tests/prism_sample_test.qml.

var LONG_TITLE = "A very long title that goes on and on to show how a row cuts its text with an ellipsis instead of growing, " + "because the row height must never change with the text it holds";

var ROWS = [
    {"id": "s01", "title": "Firefox", "subtitle": "Web browser", "hint": "App", "icon": "firefox"},
    {"id": "s02", "title": "Terminal", "subtitle": "Run commands in a shell", "hint": "App", "icon": "utilities-terminal"},
    {"id": "s03", "title": "Files", "subtitle": "Browse your folders", "hint": "App", "icon": "system-file-manager"},
    {"id": "s04", "title": "Settings", "subtitle": "Change how the system works", "hint": "App", "icon": "preferences-system"},
    {"id": "s05", "title": "Text editor", "hint": "App", "icon": "accessories-text-editor"},
    {"id": "s06", "title": "Calculator", "subtitle": "Do arithmetic", "icon": "accessories-calculator"},
    {"id": "s07", "title": "report-2026.pdf", "subtitle": "~/Documents/Work", "hint": "12 KB"},
    {"id": "s08", "title": "holiday-photo.png", "subtitle": "~/Pictures", "hint": "3.4 MB"},
    {"id": "s09", "title": "notes.md", "subtitle": "~/Documents", "hint": "1 KB"},
    {"id": "s10", "title": "invoice-march.pdf", "subtitle": "~/Documents/Money", "hint": "88 KB"},
    {"id": "s11", "title": "systemctl --user status", "subtitle": "Show the state of user services", "hint": "Command"},
    {"id": "s12", "title": "ls -la", "subtitle": "List all files with details", "hint": "Command"},
    {"id": "s13", "title": "git status", "hint": "Command"},
    {"id": "s14", "title": "Search the web for \"quickshell\"", "subtitle": "Open the results in the browser", "hint": "Web"},
    {"id": "s15", "title": "Search the web for \"wayland\"", "subtitle": "Open the results in the browser", "hint": "Web"},
    {"id": "s16", "title": "main.go:42", "subtitle": "func main() {", "hint": "Match"},
    {"id": "s17", "title": "config.json:7", "subtitle": "\"theme\": \"tokyo-night\"", "hint": "Match"},
    {"id": "s18", "title": LONG_TITLE, "subtitle": "A row with a title that is far too long for one line", "hint": "Long"},
    {"id": "s19", "title": "Music player", "subtitle": "Play your songs", "hint": "App", "icon": "multimedia-audio-player"},
    {"id": "s20", "title": "Video player", "subtitle": "Watch your videos", "hint": "App", "icon": "multimedia-video-player"},
    {"id": "s21", "title": "Archive manager", "subtitle": "Open and create archives", "hint": "App", "icon": "package-x-generic"},
    {"id": "s22", "title": "Image viewer", "hint": "App", "icon": "image-x-generic"},
    {"id": "s23", "title": "Disk usage", "subtitle": "See what takes the space", "hint": "App", "icon": "drive-harddisk"},
    {"id": "s24", "title": "System monitor", "subtitle": "Processes and resources", "hint": "App", "icon": "utilities-system-monitor"}
];

// The rows whose title contains `query`, compared without regard to case. An empty query
// keeps every row. Returns copies, each with an empty `source` (they have no supplier).
function filter(rows, query) {
    var q = query === null || query === undefined ? "" : String(query).toLowerCase();
    var out = [];
    for (var i = 0; i < rows.length; i++) {
        if (q === "" || String(rows[i].title).toLowerCase().indexOf(q) >= 0) {
            out.push({
                "id": rows[i].id,
                "title": rows[i].title,
                "subtitle": rows[i].subtitle === undefined ? "" : rows[i].subtitle,
                "hint": rows[i].hint === undefined ? "" : rows[i].hint,
                "icon": rows[i].icon === undefined ? "" : rows[i].icon,
                "source": ""
            });
        }
    }
    return out;
}
