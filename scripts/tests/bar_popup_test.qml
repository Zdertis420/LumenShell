import "../../modules/bar/popup.js" as Popup
import QtQuick
import QtTest

// Unit tests of the popup placement rule (FR-038): modules/bar/popup.js, run by qmltestrunner
// through `make test`.
TestCase {
    readonly property var screen: ({
        "x": 0,
        "y": 0,
        "width": 1000,
        "height": 600
    })

    function anchor(x, y, w, h) {
        return {
            "x": x,
            "y": y,
            "width": w,
            "height": h
        };
    }

    function size(w, h) {
        return {
            "width": w,
            "height": h
        };
    }

    function overlaps(a, b) {
        return a.x < b.x + b.width && b.x < a.x + a.width && a.y < b.y + b.height && b.y < a.y + a.height;
    }

    function inside(r, s) {
        return r.x >= s.x && r.y >= s.y && r.x + r.width <= s.x + s.width && r.y + r.height <= s.y + s.height;
    }

    function test_belowTheItemForATopBar() {
        var item = anchor(400, 0, 100, 30);
        var r = Popup.place(item, size(200, 100), screen, "top", 0.5);
        compare(r.y, 30);
        compare(r.x, 350);
        compare(r.width, 200);
        compare(r.height, 100);
    }

    function test_aboveTheItemForABottomBar() {
        var item = anchor(400, 570, 100, 30);
        var r = Popup.place(item, size(200, 100), screen, "bottom", 0.5);
        compare(r.y, 470);
        compare(r.height, 100);
        verify(!overlaps(r, item));
    }

    function test_shiftedToStayOnTheScreenAtBothSides() {
        var r = Popup.place(anchor(0, 0, 40, 30), size(300, 100), screen, "top", 0.5);
        compare(r.x, 0);
        r = Popup.place(anchor(960, 0, 40, 30), size(300, 100), screen, "top", 0.5);
        compare(r.x, 700);
        verify(inside(r, screen));
    }

    function test_sizeIsLimitedToAFractionOfTheScreen() {
        var r = Popup.place(anchor(400, 0, 100, 30), size(5000, 5000), screen, "top", 0.5);
        compare(r.width, 500);
        verify(r.height <= 300);
        verify(inside(r, screen));
    }

    function test_neverOverlapsTheItem() {
        var anchors = [anchor(0, 0, 40, 30), anchor(960, 0, 40, 30), anchor(480, 0, 100, 30)];
        for (var i = 0; i < anchors.length; i++) {
            var r = Popup.place(anchors[i], size(400, 400), screen, "top", 0.5);
            verify(!overlaps(r, anchors[i]), "popup " + i + " overlaps its item");
            verify(inside(r, screen), "popup " + i + " leaves the screen");
        }
        var bottom = anchor(480, 570, 100, 30);
        var rb = Popup.place(bottom, size(400, 400), screen, "bottom", 0.5);
        verify(!overlaps(rb, bottom));
        verify(inside(rb, screen));
    }

    function test_zeroOrNegativeSizesAreRaisedToAMinimum() {
        var r = Popup.place(anchor(400, 0, 100, 30), size(0, -5), screen, "top", 0.5);
        compare(r.width, 1);
        compare(r.height, 1);
    }

    function test_anAnchorAtAScreenCornerStillFits() {
        var corners = [anchor(0, 0, 50, 30), anchor(950, 0, 50, 30)];
        for (var i = 0; i < corners.length; i++) {
            var r = Popup.place(corners[i], size(400, 200), screen, "top", 0.5);
            verify(inside(r, screen));
        }
        var rb = Popup.place(anchor(0, 570, 50, 30), size(400, 200), screen, "bottom", 0.5);
        verify(inside(rb, screen));
    }

    function test_limitedByTheRoomOnItsSide() {
        // An item low on the screen leaves little room below for a top-style placement.
        var item = anchor(400, 560, 100, 30);
        var r = Popup.place(item, size(200, 300), screen, "top", 0.5);
        verify(r.y >= 590);
        verify(inside(r, screen));
        verify(!overlaps(r, item));
    }

    name: "BarPopup"
}
