import "../../modules/prism/rows.js" as Rows
import QtQuick
import QtTest

// Unit tests of the Prism 1.1.0 additions (specs/006-prism-modes/contracts/prism-source-slot-1.1.md):
// row validation with marks, the keep-selection rule and the message check.
TestCase {
    function row(id, extra) {
        return Object.assign({
            "id": id,
            "title": "title " + id
        }, extra ?? {
        });
    }

    function test_marksAreKeptAndMerged() {
        const r = Rows.validateRows("s", [row("1", {
            "title": "hello world",
            "marks": [[6, 9], [0, 2], [1, 3], [9, 11]]
        })]);
        compare(r.warnings.length, 0);
        compare(JSON.stringify(r.rows[0].marks), "[[0,3],[6,11]]");
    }

    function test_badMarksAreDroppedWithAWarning() {
        const r = Rows.validateRows("src", [row("1", {
            "title": "abc",
            "marks": [[0, 1], [2, 2], [-1, 1], [1, 9], ["a", 1], [1], 7]
        })]);
        compare(JSON.stringify(r.rows[0].marks), "[[0,1]]");
        compare(r.warnings.length, 6);
        compare(r.warnings[0].place, "src");
        verify(r.warnings[0].problem.indexOf("mark") >= 0);
    }

    function test_marksThatIsNotAListIsIgnored() {
        const r = Rows.validateRows("s", [row("1", {
            "marks": "no"
        })]);
        compare(r.rows[0].marks.length, 0);
        compare(r.warnings.length, 1);
    }

    function test_marksCountRunesNotUtf16Units() {
        // The emoji is two UTF-16 units but one rune; the mark on it must be accepted.
        const r = Rows.validateRows("s", [row("1", {
            "title": "a\u{1F600}b",
            "marks": [[1, 2]]
        })]);
        compare(r.warnings.length, 0);
        compare(JSON.stringify(r.rows[0].marks), "[[1,2]]");
    }

    function test_marksBeyondTheCutTitleAreDropped() {
        const wide = new Array(301).join("x");
        const r = Rows.validateRows("s", [row("1", {
            "title": wide,
            "marks": [[250, 260]]
        })]);
        compare(r.rows[0].title.length, 200);
        compare(r.rows[0].marks.length, 0);
        compare(r.warnings.length, 1);
    }

    function test_rowRulesOfFeatureFiveStillHold() {
        const r = Rows.validateRows("s", [row("1"), row("1"), {
            "title": "no id"
        }, {
            "id": "x"
        }, null, row("2", {
            "subtitle": "s",
            "hint": "h",
            "icon": "i"
        })]);
        compare(r.rows.length, 2);
        compare(r.warnings.length, 4);
        compare(r.rows[1].subtitle, "s");
        compare(r.rows[1].source, "s");
        verify(Rows.validateRows("s", "nope").warnings.length === 1);
    }

    function test_rowLimit() {
        const list = [];
        for (let i = 0; i < 1005; i++) list.push(row("r" + i))
        const r = Rows.validateRows("s", list);
        compare(r.rows.length, 1000);
        compare(r.warnings.length, 1);
    }

    function entries(ids) {
        return ids.map((id) => {
            return {
                "id": id,
                "source": "s"
            };
        });
    }

    function test_selectionStaysOnTheSameRow() {
        // Rows arrive below the selected row: the index stays.
        compare(Rows.selectionAfter(entries(["a", "b"]), 1, entries(["a", "b", "c", "d"]), true), 1);
        // A row appears above it: the selection follows the row, not the index.
        compare(Rows.selectionAfter(entries(["b", "c"]), 0, entries(["a", "b", "c"]), true), 1);
    }

    function test_selectionWhenTheRowIsGone() {
        compare(Rows.selectionAfter(entries(["a", "b", "c"]), 2, entries(["a", "b"]), true), 1);
        compare(Rows.selectionAfter(entries(["a", "b"]), 1, [], true), -1);
    }

    function test_withoutKeepTheFirstRowIsSelected() {
        compare(Rows.selectionAfter(entries(["a", "b"]), 1, entries(["a", "b"]), false), 0);
        compare(Rows.selectionAfter([], -1, entries(["a"]), true), 0);
    }

    function test_selectionDistinguishesSources() {
        const was = [{
            "id": "1",
            "source": "x"
        }];
        const now = [{
            "id": "1",
            "source": "y"
        }, {
            "id": "1",
            "source": "x"
        }];
        compare(Rows.selectionAfter(was, 0, now, true), 1);
    }

    function test_messageCheck() {
        compare(JSON.stringify(Rows.checkMessage("hi", "error")), "{\"text\":\"hi\",\"kind\":\"error\"}");
        compare(Rows.checkMessage("hi", "weird").kind, "info");
        compare(Rows.checkMessage(5, "info").text, "");
        compare(Rows.checkMessage(new Array(301).join("y"), "info").text.length, 200);
    }

}
