.pragma library

// Prism's mode prefix (specs/005-prism-window, data-model.md "Mode" and research.md R10).
// Pure functions, no QML types, so they are unit tested without a shell
// (scripts/tests/prism_prefix_test.qml).
//
// The text starts with "@", one configured letter, and then a space or the end of the text:
// "@f report" is the mode "files" with the query "report". Anything else is the mode
// "default" and the whole text is the query.

// The modes in the order of the settings, with the setting of each one's letter and the
// letter it has by default. The mode names are what modules receive (not the letters).
var MODES = [
    {"mode": "apps", "key": "prefixApps", "letter": "a"},
    {"mode": "files", "key": "prefixFiles", "letter": "f"},
    {"mode": "web", "key": "prefixWeb", "letter": "w"},
    {"mode": "commands", "key": "prefixCommands", "letter": "s"},
    {"mode": "grep", "key": "prefixGrep", "letter": "g"}
];

var SIGN = "@";
var SIGN_AND_LETTER = 2; // the length of "@f"
var SPACE = " ";

// The default letters by mode name.
function defaultLetters() {
    var out = {};
    for (var i = 0; i < MODES.length; i++)
        out[MODES[i].mode] = MODES[i].letter;
    return out;
}

function isLetter(c) {
    return typeof c === "string" && c.length === 1 && ((c >= "a" && c <= "z") || (c >= "0" && c <= "9"));
}

// Splits `text` by the letters ({apps, files, web, commands, grep}; a missing or null letter
// means that mode has no prefix). Returns {mode, query, prefixLength}; prefixLength is the
// length of the "@x" that was recognized, or 0.
function parse(text, letters) {
    var t = text === null || text === undefined ? "" : String(text);
    var plain = {
        "mode": "default",
        "query": t,
        "prefixLength": 0
    };
    if (t.length < SIGN_AND_LETTER || t.charAt(0) !== SIGN)
        return plain;
    if (t.length > SIGN_AND_LETTER && t.charAt(SIGN_AND_LETTER) !== SPACE)
        return plain;
    var letter = t.charAt(1);
    for (var i = 0; i < MODES.length; i++) {
        var wanted = letters ? letters[MODES[i].mode] : null;
        if (wanted !== null && wanted !== undefined && wanted === letter) {
            return {
                "mode": MODES[i].mode,
                "query": t.length > SIGN_AND_LETTER ? t.substring(SIGN_AND_LETTER + 1) : "",
                "prefixLength": SIGN_AND_LETTER
            };
        }
    }
    return plain;
}

// Resolves the five letter settings (`settings` has prefixApps ... prefixGrep) into
// {letters, errors}. In the order of MODES a letter that is not one lowercase letter or digit
// or that an earlier mode already uses falls back to its default; when the default is taken
// too, that mode has no prefix (its letter is null). Each fallback is an error
// {key, problem, fallback} that names the key.
function checkLetters(settings) {
    var s = settings || {};
    var letters = {};
    var errors = [];
    var used = {};
    for (var i = 0; i < MODES.length; i++) {
        var m = MODES[i];
        var value = s[m.key];
        if (value === null || value === undefined)
            value = m.letter;
        var problem = "";
        if (!isLetter(value))
            problem = "must be one lowercase letter or digit";
        else if (used[value])
            problem = "\"" + value + "\" is already the prefix of another mode";
        if (problem === "") {
            letters[m.mode] = value;
            used[value] = true;
            continue;
        }
        var fallback = used[m.letter] ? null : m.letter;
        errors.push({
            "key": m.key,
            "problem": problem,
            "fallback": fallback
        });
        letters[m.mode] = fallback;
        if (fallback !== null)
            used[fallback] = true;
    }
    return {
        "letters": letters,
        "errors": errors
    };
}
