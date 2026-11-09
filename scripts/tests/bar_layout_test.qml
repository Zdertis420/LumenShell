import "../../modules/bar/layout.js" as Layout
import QtQuick
import QtTest

// Unit tests of the bar's fitting rule (FR-015a): modules/bar/layout.js, run by qmltestrunner
// through `make test`.
TestCase {
    function fit(left, center, right, available, gap) {
        return Layout.fit({
            "left": left,
            "center": center,
            "right": right
        }, available, gap);
    }

    function names(dropped) {
        return dropped.map(function(d) {
            return d.place + ":" + d.index;
        }).join(",");
    }

    function test_everythingFits() {
        var r = fit([100], [100], [100, 100], 1000, 10);
        compare(r.left, 1);
        compare(r.center, 1);
        compare(r.right, 2);
        compare(r.dropped.length, 0);
    }

    function test_oneItemDroppedFromTheEndOfTheRight() {
        // 100 + 100 + 100 + 100 = 400 plus gaps: 3 inside regions or between regions = 30.
        var r = fit([100], [100], [100, 100], 420, 10);
        compare(r.right, 1);
        compare(r.left, 1);
        compare(r.center, 1);
        compare(names(r.dropped), "right:1");
    }

    function test_rightIsExhaustedBeforeTheCenter() {
        var r = fit([100], [100], [100, 100], 230, 10);
        compare(names(r.dropped), "right:1,right:0");
        compare(r.right, 0);
        compare(r.center, 1);
        compare(r.left, 1);
    }

    function test_thenTheCenterThenTheLeft() {
        var r = fit([100, 100], [100, 100], [100], 150, 10);
        // Drop order: right first, then the end of the center, then the end of the left.
        compare(names(r.dropped), "right:0,center:1,center:0,left:1");
        compare(r.left, 1);
        compare(r.center, 0);
        compare(r.right, 0);
    }

    function test_allDroppedWhenThereIsNoRoom() {
        var r = fit([50], [50], [50], 0, 10);
        compare(r.left + r.center + r.right, 0);
        compare(r.dropped.length, 3);
        r = fit([50], [50], [50], -5, 10);
        compare(r.left + r.center + r.right, 0);
    }

    function test_wideningBringsItemsBackInTheOppositeOrder() {
        var widths = [[100, 100], [100], [100, 100]];
        var order = [];
        for (var available = 10; available <= 700; available += 5) {
            var r = fit(widths[0], widths[1], widths[2], available, 10);
            order.push(r.left + r.center + r.right);
        }
        for (var i = 1; i < order.length; i++) verify(order[i] >= order[i - 1], "the number of kept items never shrinks when the bar widens")
        compare(order[order.length - 1], 5);
    }

    function test_gapsCountBetweenItemsAndBetweenRegions() {
        // Two regions with one item each: 100 + gap + 100 = 210 with a gap of 10.
        compare(fit([100], [], [100], 210, 10).dropped.length, 0);
        compare(fit([100], [], [100], 209, 10).dropped.length, 1);
        // Two items in one region: 100 + gap + 100.
        compare(fit([], [], [100, 100], 210, 10).dropped.length, 0);
        compare(fit([], [], [100, 100], 209, 10).dropped.length, 1);
    }

    function test_emptyLists() {
        var r = fit([], [], [], 500, 10);
        compare(r.left + r.center + r.right, 0);
        compare(r.dropped.length, 0);
    }

    function test_aSingleItemWiderThanTheScreenIsDropped() {
        var r = fit([], [900], [], 500, 10);
        compare(r.center, 0);
        compare(names(r.dropped), "center:0");
    }

    function test_zeroWidthItemsTakeNoRoomAndAreNotReported() {
        // The first right item is hidden (width 0): it is neither counted nor reported.
        var r = fit([100], [], [0, 100], 210, 10);
        compare(r.dropped.length, 0);
        r = fit([100], [], [0, 100], 200, 10);
        compare(names(r.dropped), "right:1");
        // Dropping reaches past a hidden item without reporting it.
        r = fit([100], [], [100, 0], 100, 10);
        compare(names(r.dropped), "right:0");
        compare(r.right, 0);
    }

    function test_noItemIsDroppedTwice() {
        var r = fit([100, 100], [100, 100], [100, 100], 250, 10);
        var seen = {
        };
        for (var i = 0; i < r.dropped.length; i++) {
            var key = r.dropped[i].place + ":" + r.dropped[i].index;
            verify(!seen[key], key + " dropped twice");
            seen[key] = true;
        }
    }

    function test_centerIsCenteredWhenItFits() {
        var r = fit([100], [100], [100], 1000, 10);
        compare(r.centerX, 450);
    }

    function test_centerMovesBetweenTheOthersWhenCenteringDoesNotFit() {
        // Left is wide, so a centered center would overlap it: it sits after the left region.
        var r = fit([400], [100], [], 1000, 10);
        verify(r.centerX >= 410, "after the left region and its gap: " + r.centerX);
        verify(r.centerX + 100 <= 1000, "inside the bar");
        // Right is wide.
        r = fit([], [100], [400], 1000, 10);
        verify(r.centerX + 100 <= 590, "before the right region and its gap: " + r.centerX);
        verify(r.centerX >= 0);
    }

    function test_centerXIsZeroWithoutACenterRegion() {
        compare(fit([100], [], [100], 1000, 10).centerX, 0);
    }

    // ---- specs/004-bar-customization-styles: two gaps, place padding, any axis ----
    function fitOpts(left, center, right, available, opts) {
        return Layout.fit({
            "left": left,
            "center": center,
            "right": right
        }, available, opts);
    }

    function test_aNumberMeansOneGapForItemsAndPlaces() {
        var a = fit([100, 100], [100], [100], 500, 10);
        var b = fitOpts([100, 100], [100], [100], 500, {
            "itemGap": 10,
            "placeGap": 10,
            "placePadding": 0
        });
        compare(a.left, b.left);
        compare(a.center, b.center);
        compare(a.right, b.right);
        compare(a.centerX, b.centerX);
        compare(names(a.dropped), names(b.dropped));
    }

    function test_itemGapAndPlaceGapAreCountedSeparately() {
        // left: 100 + 100 + item gap 4 = 204; right: 100; one place gap of 20 between them.
        var opts = {
            "itemGap": 4,
            "placeGap": 20,
            "placePadding": 0
        };
        compare(Layout.totalWidth({
            "left": [100, 100],
            "center": [],
            "right": [100]
        }, {
            "left": 2,
            "center": 0,
            "right": 1
        }, opts), 324);
        // 323 does not fit, so the last item of the right place goes.
        var r = fitOpts([100, 100], [], [100], 323, opts);
        compare(names(r.dropped), "right:0");
        r = fitOpts([100, 100], [], [100], 324, opts);
        compare(r.dropped.length, 0);
    }

    function test_placePaddingIsAddedToEachNonEmptyPlace() {
        var opts = {
            "itemGap": 0,
            "placeGap": 10,
            "placePadding": 5
        };
        // Two places of one 100 item: (100 + 10) * 2 + gap 10 = 230. An empty place adds nothing.
        compare(Layout.totalWidth({
            "left": [100],
            "center": [],
            "right": [100]
        }, {
            "left": 1,
            "center": 0,
            "right": 1
        }, opts), 230);
        compare(Layout.widthOf({
            "left": [100],
            "center": [],
            "right": []
        }, {
            "left": 1,
            "center": 0,
            "right": 0
        }, "center", opts), 0);
    }

    function test_placePaddingAffectsWhatFits() {
        var opts = {
            "itemGap": 0,
            "placeGap": 10,
            "placePadding": 5
        };
        var r = fitOpts([100], [], [100], 229, opts);
        compare(names(r.dropped), "right:0");
        r = fitOpts([100], [], [100], 230, opts);
        compare(r.dropped.length, 0);
    }

    function test_theSameRuleWorksOnALengthTakenFromAHeight() {
        // A vertical bar on a 1080 px tall screen: item lengths are heights.
        var r = fitOpts([40, 40], [60], [40, 40, 40], 1080, {
            "itemGap": 8,
            "placeGap": 8,
            "placePadding": 0
        });
        compare(r.dropped.length, 0);
        r = fitOpts([300, 300], [300], [300, 300], 1080, {
            "itemGap": 8,
            "placeGap": 8,
            "placePadding": 0
        });
        compare(names(r.dropped), "right:1,right:0");
    }

    function test_centerUsesThePlaceGapNextToTheOtherPlaces() {
        var r = fitOpts([400], [100], [], 1000, {
            "itemGap": 0,
            "placeGap": 30,
            "placePadding": 0
        });
        verify(r.centerX >= 430, "after the left place and its place gap: " + r.centerX);
    }

    name: "BarLayout"
}
