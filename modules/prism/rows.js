.pragma library

// The rules for the rows a source supplies to Prism, and for keeping the selection when rows
// change (specs/005-prism-window data-model.md "Row"; specs/006-prism-modes
// contracts/prism-source-slot-1.1.md). Pure functions, unit tested without a shell
// (scripts/tests/prism_slot_1_1_test.qml).

var ROW_LIMIT = 1000; // documented constant: rows a source may supply
var TEXT_LIMIT = 200; // documented constant: characters of a title, subtitle, hint or message
var ID_LIMIT = 200; // documented constant: characters of a row id

function clip(value) {
    return String(value).slice(0, TEXT_LIMIT);
}

// Checks the `marks` of a row: a list of [start, end] rune offsets in the title, end exclusive.
// Returns {marks, problems}; a range that is not two integers with 0 <= start < end <= title
// length is dropped, and overlapping ranges are merged so the window can draw them in order.
function checkMarks(marks, titleLength) {
    var out = {
        "marks": [],
        "problems": []
    };
    if (marks === undefined || marks === null)
        return out;
    if (!Array.isArray(marks)) {
        out.problems.push("marks must be a list of [start, end] ranges; ignored");
        return out;
    }
    var good = [];
    for (var i = 0; i < marks.length; i++) {
        var m = marks[i];
        var ok = Array.isArray(m) && m.length === 2 && Number.isInteger(m[0]) && Number.isInteger(m[1]) && m[0] >= 0 && m[0] < m[1] && m[1] <= titleLength;
        if (!ok) {
            out.problems.push("mark " + i + " is not a range inside the title; dropped");
            continue;
        }
        good.push([m[0], m[1]]);
    }
    good.sort(function (a, b) {
        return a[0] - b[0];
    });
    for (var j = 0; j < good.length; j++) {
        var last = out.marks[out.marks.length - 1];
        if (last && good[j][0] <= last[1])
            last[1] = Math.max(last[1], good[j][1]);
        else
            out.marks.push(good[j]);
    }
    return out;
}

// Checks the list a source gave to setRows. Returns {rows, warnings}; a row that breaks a rule
// is dropped with a warning that names the source (the `place`).
function validateRows(source, list) {
    var out = {
        "rows": [],
        "warnings": []
    };
    function warn(index, entry, problem) {
        out.warnings.push({
            "place": source,
            "index": index,
            "entry": entry === null || entry === undefined ? null : {
                "id": entry.id
            },
            "problem": problem
        });
    }
    if (!Array.isArray(list)) {
        warn(0, null, "setRows needs a list of rows; ignored");
        return out;
    }
    var seen = {};
    for (var i = 0; i < list.length; i++) {
        var r = list[i];
        if (out.rows.length >= ROW_LIMIT) {
            warn(i, null, "more than " + ROW_LIMIT + " rows; the rest are dropped");
            break;
        }
        if (r === null || typeof r !== "object") {
            warn(i, null, "a row must be an object; dropped");
            continue;
        }
        if (typeof r.id !== "string" || r.id === "" || r.id.length > ID_LIMIT) {
            warn(i, r, "a row needs an id of 1 to " + ID_LIMIT + " characters; dropped");
            continue;
        }
        if (typeof r.title !== "string" || r.title === "") {
            warn(i, r, "a row needs a title; dropped");
            continue;
        }
        if (seen[r.id]) {
            warn(i, r, "duplicate id; dropped");
            continue;
        }
        seen[r.id] = true;
        var title = clip(r.title);
        var checked = checkMarks(r.marks, Array.from(title).length);
        for (var p = 0; p < checked.problems.length; p++)
            warn(i, r, checked.problems[p]);
        out.rows.push({
            "id": r.id,
            "title": title,
            "subtitle": r.subtitle === undefined || r.subtitle === null ? "" : clip(r.subtitle),
            "hint": r.hint === undefined || r.hint === null ? "" : clip(r.hint),
            "icon": typeof r.icon === "string" ? r.icon : "",
            "marks": checked.marks,
            "source": source
        });
    }
    return out;
}

// Which row stays selected after the rows changed (the keepSelection argument of setRows).
// `oldRows` and `newRows` are lists of rows with `id` and `source`; `oldIndex` is the selected
// index in oldRows. Returns the new index: the same row by source and id when it is still there,
// otherwise the nearest index, or -1 when there are no rows. Without `keep` the first row is
// selected, as in Feature V.
function selectionAfter(oldRows, oldIndex, newRows, keep) {
    if (newRows.length === 0)
        return -1;
    if (!keep || oldIndex < 0 || oldIndex >= oldRows.length)
        return 0;
    var was = oldRows[oldIndex];
    for (var i = 0; i < newRows.length; i++) {
        if (newRows[i].id === was.id && newRows[i].source === was.source)
            return i;
    }
    return Math.min(oldIndex, newRows.length - 1);
}

// Checks the arguments of setMessage. Returns {text, kind}: an unknown kind is `info`, a text
// that is not a string is empty, and the text is cut.
function checkMessage(text, kind) {
    return {
        "text": typeof text === "string" ? clip(text) : "",
        "kind": kind === "error" ? "error" : "info"
    };
}
