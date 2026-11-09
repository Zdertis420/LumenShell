import "../../modules/prism/sample.js" as Sample
import QtQuick
import QtTest

// Unit tests of the preview rows (FR-015): modules/prism/sample.js.
TestCase {
    function test_thereAre24Rows() {
        compare(Sample.ROWS.length, 24);
    }

    function test_idsAreUniqueAndTitlesExist() {
        const seen = {
        };
        for (const r of Sample.ROWS) {
            verify(r.id && r.title, "id and title");
            verify(!seen[r.id], "duplicate id " + r.id);
            seen[r.id] = true;
        }
    }

    function test_oneTitleIsOver100Characters() {
        verify(Sample.ROWS.some((r) => {
            return r.title.length > 100;
        }));
    }

    function test_rowsDifferInTheirParts() {
        verify(Sample.ROWS.some((r) => {
            return r.subtitle === undefined;
        }));
        verify(Sample.ROWS.some((r) => {
            return r.hint === undefined;
        }));
        verify(Sample.ROWS.some((r) => {
            return r.icon === undefined;
        }));
        verify(Sample.ROWS.some((r) => {
            return r.icon !== undefined;
        }));
    }

    function test_emptyQueryKeepsEveryRow() {
        compare(Sample.filter(Sample.ROWS, "").length, 24);
        compare(Sample.filter(Sample.ROWS, null).length, 24);
    }

    function test_filterIsAContainsMatchWithoutCase() {
        const r = Sample.filter(Sample.ROWS, "FIRE");
        compare(r.length, 1);
        compare(r[0].title, "Firefox");
        verify(Sample.filter(Sample.ROWS, "player").length === 2);
    }

    function test_noMatchGivesAnEmptyList() {
        compare(Sample.filter(Sample.ROWS, "zzzz-nothing").length, 0);
    }

    function test_filteredRowsHaveStringsForEveryPart() {
        for (const r of Sample.filter(Sample.ROWS, "")) {
            compare(typeof r.subtitle, "string");
            compare(typeof r.hint, "string");
            compare(typeof r.icon, "string");
        }
    }

    function test_filteredRowsHaveNoSource() {
        for (const r of Sample.filter(Sample.ROWS, "")) compare(r.source, "")
    }

}
