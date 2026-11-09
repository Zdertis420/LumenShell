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

    // ---- specs/004-bar-customization-styles: left and right edges ----
    function test_rightOfTheItemForALeftBar() {
        var item = anchor(0, 200, 37, 60);
        var r = Popup.place(item, size(200, 100), screen, "left", 0.5);
        compare(r.x, 37);
        compare(r.y, 180, "centered on the item along the bar");
        compare(r.width, 200);
        compare(r.height, 100);
        verify(!overlaps(r, item));
    }

    function test_leftOfTheItemForARightBar() {
        var item = anchor(963, 200, 37, 60);
        var r = Popup.place(item, size(200, 100), screen, "right", 0.5);
        compare(r.x, 763);
        compare(r.width, 200);
        compare(r.y, 180);
        verify(!overlaps(r, item));
    }

    function test_shiftedToStayOnTheScreenAlongAVerticalBar() {
        // Near the top and near the bottom of the screen the popup moves, and stays inside.
        var top = Popup.place(anchor(0, 0, 37, 30), size(200, 300), screen, "left", 0.5);
        compare(top.y, 0);
        verify(inside(top, screen));
        var bottom = Popup.place(anchor(0, 570, 37, 30), size(200, 300), screen, "left", 0.5);
        compare(bottom.y, 300);
        verify(inside(bottom, screen));
    }

    function test_limitedByTheRoomOnItsSideOfAVerticalBar() {
        // A right bar item near the left of the screen leaves little room before it.
        var item = anchor(40, 200, 37, 60);
        var r = Popup.place(item, size(300, 100), screen, "right", 0.5);
        verify(r.width <= 40);
        verify(inside(r, screen));
        verify(!overlaps(r, item));
        // A left bar item near the right edge of the screen leaves little room after it.
        item = anchor(960, 200, 37, 60);
        r = Popup.place(item, size(300, 100), screen, "left", 0.5);
        verify(inside(r, screen));
        verify(!overlaps(r, item));
    }

    function test_neverOverlapsTheItemOnAnyEdge() {
        var edges = ["top", "bottom", "left", "right"];
        var items = [anchor(400, 0, 100, 30), anchor(400, 570, 100, 30), anchor(0, 200, 37, 60), anchor(963, 200, 37, 60)];
        for (var i = 0; i < edges.length; i++) {
            var r = Popup.place(items[i], size(5000, 5000), screen, edges[i], 0.5);
            verify(!overlaps(r, items[i]), edges[i]);
            verify(inside(r, screen), edges[i]);
            verify(r.width <= 500 && r.height <= 300, edges[i] + " limited to half of the screen");
        }
    }

    name: "BarPopup"
}
