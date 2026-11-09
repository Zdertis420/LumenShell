import "../../modules/bar/geometry.js" as Geometry
import QtQuick
import QtTest

// Unit tests of the bar's sizes: modules/bar/geometry.js (defaults per style, limits for small
// screens), run by qmltestrunner through `make test`.
// Contracts: specs/004-bar-customization-styles/contracts/bar-config-1.2.md and bar-surfaces.md.
TestCase {
    readonly property var tokens: ({
        "spacing": {
            "xs": 4,
            "sm": 8,
            "md": 12,
            "lg": 16,
            "xl": 24
        },
        "radius": {
            "sm": 4,
            "md": 8,
            "lg": 16,
            "full": 9999
        },
        "font": {
            "body": 13
        }
    })
    readonly property var screen: ({
        "width": 1920,
        "height": 1080
    })

    function resolve(settings, scr) {
        return Geometry.resolve(settings, tokens, scr || screen);
    }

    function keys(warnings) {
        return warnings.map(function(w) {
            return w.key;
        }).join(",");
    }

    // ---- defaults ----
    function test_solidDefaultsAreTheBarOfFeatureIII() {
        var r = resolve({
        });
        var s = r.sizes;
        compare(s.thickness, 37, "13 + 2 x 12");
        compare(s.margin, 0);
        compare(s.padding, 8);
        compare(s.itemGap, 8);
        compare(s.placeGap, 8);
        compare(s.radius, 0);
        compare(s.itemPadding, 12);
        compare(s.itemRadius, 4);
        compare(s.popupRadius, 8);
        compare(s.popupPadding, 0);
        compare(s.popupBorderWidth, 1);
        compare(s.reserved.top, 37);
        compare(s.reserved.bottom, 0);
        compare(s.reserved.left, 0);
        compare(s.reserved.right, 0);
        compare(r.warnings.length, 0);
    }

    function test_nullIsTheSameAsUnset() {
        var r = resolve({
            "thickness": null,
            "margin": null,
            "fontSize": null,
            "radius": null
        });
        compare(r.sizes.thickness, 37);
        compare(r.sizes.margin, 0);
        compare(r.sizes.radius, 0);
    }

    function test_islandsDefaults() {
        var s = resolve({
            "style": "islands"
        }).sizes;
        compare(s.margin, 8);
        compare(s.padding, 12);
        compare(s.radius, 9999, "fully round");
        compare(s.reserved.top, 45, "thickness 37 + margin 8");
    }

    function test_frameDefaults() {
        var r = resolve({
            "style": "frame"
        });
        var s = r.sizes;
        compare(s.margin, 0);
        compare(s.frameBorder, 8);
        compare(s.frameRadius, 16);
        compare(s.reserved.top, 37);
        compare(s.reserved.bottom, 8);
        compare(s.reserved.left, 8);
        compare(s.reserved.right, 8);
        compare(r.warnings.length, 0);
    }

    function test_theThicknessFollowsTheFontSize() {
        compare(resolve({
            "fontSize": 20
        }).sizes.thickness, 44);
        compare(resolve({
            "fontSize": 20,
            "thickness": 60
        }).sizes.thickness, 60, "a set thickness wins");
    }

    // ---- edges ----
    function test_theEdgeDecidesWhereTheSpaceIsReserved() {
        var cases = {
            "top": ["top", 37, 0, 0, 0],
            "bottom": ["bottom", 0, 37, 0, 0],
            "left": ["left", 0, 0, 37, 0],
            "right": ["right", 0, 0, 0, 37]
        };
        for (var edge in cases) {
            var s = resolve({
                "edge": edge
            }).sizes;
            compare(s.reserved.top, cases[edge][1], edge);
            compare(s.reserved.bottom, cases[edge][2], edge);
            compare(s.reserved.left, cases[edge][3], edge);
            compare(s.reserved.right, cases[edge][4], edge);
            compare(s.vertical, edge === "left" || edge === "right");
        }
    }

    function test_theFrameKeepsTheBorderOnTheOtherSides() {
        var s = resolve({
            "style": "frame",
            "edge": "left",
            "frameBorder": 10
        }).sizes;
        compare(s.reserved.left, 37);
        compare(s.reserved.top, 10);
        compare(s.reserved.bottom, 10);
        compare(s.reserved.right, 10);
    }

    function test_aFrameBorderOfZeroRemovesTheOtherSidesButKeepsTheCorners() {
        var s = resolve({
            "style": "frame",
            "frameBorder": 0
        }).sizes;
        compare(s.reserved.bottom, 0);
        compare(s.reserved.left, 0);
        compare(s.reserved.right, 0);
        compare(s.reserved.top, 37);
        compare(s.frameRadius, 16, "the inner corners stay");
    }

    function test_unknownValuesFallBackToSolidAndTop() {
        var s = resolve({
            "style": "pills",
            "edge": "middle"
        }).sizes;
        compare(s.style, "solid");
        compare(s.edge, "top");
    }

    // ---- explicit values ----
    function test_explicitValuesAreUsedAsGiven() {
        var s = resolve({
            "thickness": 50,
            "margin": 6,
            "padding": 3,
            "itemGap": 2,
            "placeGap": 20,
            "itemPadding": 5,
            "itemRadius": 9,
            "radius": 14,
            "popupRadius": 20,
            "popupPadding": 7,
            "popupBorderWidth": 3
        }).sizes;
        compare(s.thickness, 50);
        compare(s.margin, 6);
        compare(s.padding, 3);
        compare(s.itemGap, 2);
        compare(s.placeGap, 20);
        compare(s.itemPadding, 5);
        compare(s.itemRadius, 9);
        compare(s.radius, 14);
        compare(s.popupRadius, 20);
        compare(s.popupPadding, 7);
        compare(s.popupBorderWidth, 3);
        compare(s.reserved.top, 56, "thickness + margin");
    }

    function test_aZeroIsAValueNotUnset() {
        var s = resolve({
            "margin": 0,
            "padding": 0,
            "itemGap": 0,
            "radius": 0,
            "style": "islands"
        }).sizes;
        compare(s.margin, 0);
        compare(s.padding, 0);
        compare(s.itemGap, 0);
        compare(s.radius, 0);
    }

    // ---- limits ----
    function test_theBarMayTakeAtMostHalfOfTheScreen() {
        // A 300 px tall screen: half is 150. thickness 200 is cut to 150 (and warned about).
        var r = resolve({
            "thickness": 200
        }, {
            "width": 400,
            "height": 300
        });
        compare(r.sizes.thickness, 150);
        compare(keys(r.warnings), "thickness");
    }

    function test_marginIsCutAfterThicknessAtItsMinimum() {
        // Half of 100 is 50. thickness 40 + margin 30 = 70: thickness goes first, to 20 (30 cut
        // would go below the 16 minimum, so it stops at 16 and the margin takes the rest).
        var r = resolve({
            "thickness": 40,
            "margin": 30
        }, {
            "width": 400,
            "height": 100
        });
        compare(r.sizes.thickness + r.sizes.margin <= 50, true);
        compare(r.sizes.thickness >= 16, true);
    }

    function test_theFrameBorderCountsAcrossTheBar() {
        // 1080 tall: half is 540. thickness 500 + margin 0 + border 100 = 600: thickness is cut by 60.
        var r = resolve({
            "style": "frame",
            "thickness": 500,
            "frameBorder": 100
        });
        compare(r.sizes.thickness, 440);
        compare(r.sizes.frameBorder, 100);
        compare(keys(r.warnings), "thickness");
    }

    function test_marginsAlongTheBarAreLimitedToo() {
        // 400 wide: half is 200. 2 x margin 128 = 256 > 200: margin goes to 100.
        var r = resolve({
            "margin": 128
        }, {
            "width": 400,
            "height": 2000
        });
        compare(r.sizes.margin, 100);
        verify(keys(r.warnings).indexOf("margin") >= 0);
    }

    function test_theFrameRadiusIsLimitedToHalfOfTheInnerArea() {
        // 1920 x 1080 frame: inner area 1904 x 1035, so at most 517.
        var r = resolve({
            "style": "frame",
            "frameRadius": 600
        });
        compare(r.sizes.frameRadius, 517);
        compare(keys(r.warnings), "frameRadius");
        compare(resolve({
            "style": "frame",
            "frameRadius": 500
        }).sizes.frameRadius, 500);
    }

    function test_theFrameRadiusIsNotCheckedOutsideTheFrameStyle() {
        var r = resolve({
            "style": "solid",
            "frameRadius": 600
        });
        compare(r.warnings.length, 0);
    }

    function test_largeRadiiAreNeverReported() {
        var r = resolve({
            "radius": 256,
            "itemRadius": 64,
            "popupRadius": 64
        });
        compare(r.warnings.length, 0);
        compare(r.sizes.radius, 256);
    }

    function test_roundRadiusIsHalfOfTheSmallerSideAtMost() {
        compare(Geometry.roundRadius(9999, 200, 40), 20);
        compare(Geometry.roundRadius(5, 200, 40), 5);
        compare(Geometry.roundRadius(-3, 200, 40), 0);
    }

    function test_aVeryTinyScreenStillGivesUsableNumbers() {
        var r = resolve({
            "style": "frame",
            "thickness": 256,
            "margin": 128,
            "frameBorder": 128,
            "frameRadius": 256
        }, {
            "width": 120,
            "height": 80
        });
        var s = r.sizes;
        verify(s.thickness >= 16);
        verify(s.margin >= 0 && s.frameBorder >= 0 && s.frameRadius >= 0);
        verify(s.thickness + s.margin + s.frameBorder <= 40, "at most half of the height");
        verify(r.warnings.length > 0);
    }

    // ---- corner blocks of the frame (research R4a) ----
    function blocks(settings, scr) {
        var s = resolve(settings, scr).sizes;
        var by = {
        };
        Geometry.cornerBlocks(s).forEach(function(b) {
            by[b.id] = b;
        });
        return by;
    }

    function test_noBlocksOutsideTheFrameStyle() {
        compare(Geometry.cornerBlocks(resolve({
        }).sizes).length, 0);
        compare(Geometry.cornerBlocks(resolve({
            "style": "islands"
        }).sizes).length, 0);
    }

    function test_theFrameHasFourBlocksThatCoverTheCornerSquares() {
        // Bottom edge: the strip takes 37 at the bottom, the borders 8, the radius is 16.
        var b = blocks({
            "style": "frame",
            "edge": "bottom"
        });
        compare(Object.keys(b).length, 4);
        compare(b.tl.width, 24, "left border 8 + radius 16");
        compare(b.tl.height, 24, "top border 8 + radius 16");
        compare(b.tr.width, 24);
        compare(b.bl.width, 24);
        compare(b.bl.height, 53, "strip 37 + radius 16");
        compare(b.br.height, 53);
    }

    // The area of a polygon given as points {x, y} (shoelace formula).
    function area(points) {
        var sum = 0;
        for (var i = 0; i < points.length; i++) {
            var p = points[i];
            var q = points[(i + 1) % points.length];
            sum += p.x * q.y - q.x * p.y;
        }
        return Math.abs(sum) / 2;
    }

    function test_theBandIsTheBlockWithoutItsInnerSquare() {
        var b = blocks({
            "style": "frame",
            "edge": "bottom"
        });
        for (var id in b) {
            var k = b[id];
            compare(k.band.length, 6, id);
            compare(area(k.band), k.width * k.height - k.radius * k.radius, id + ": block minus the r x r inner square");
            // Every point is inside the block.
            k.band.forEach(function(p) {
                verify(p.x >= 0 && p.x <= k.width && p.y >= 0 && p.y <= k.height, id + " band point inside");
            });
        }
    }

    function test_theBandTouchesTheScreenCorner() {
        // The screen corner of each block (top left, top right, bottom left, bottom right) is a
        // point of its band, so the frame has no gap there.
        function has(k, x, y) {
            return k.band.some(function(p) {
                return p.x === x && p.y === y;
            });
        }

        var b = blocks({
            "style": "frame"
        });
        verify(has(b.tl, 0, 0));
        verify(has(b.tr, b.tr.width, 0));
        verify(has(b.bl, 0, b.bl.height));
        verify(has(b.br, b.br.width, b.br.height));
    }

    function test_theSpandrelSitsAtTheInnerCornerAndTheArcDirectionFollowsTheMirroring() {
        var b = blocks({
            "style": "frame"
        });
        // Top left, bar on top: reserved left 8, top 37, radius 16.
        compare(b.tl.p0.x, 8);
        compare(b.tl.p0.y, 37);
        compare(b.tl.p1.x, 24);
        compare(b.tl.p1.y, 37);
        compare(b.tl.p2.x, 8);
        compare(b.tl.p2.y, 53);
        compare(b.tl.clockwise, false);
        // Mirrored across the vertical axis only: top right. Mirrored across the horizontal axis only: bottom left.
        compare(b.tr.clockwise, true);
        compare(b.bl.clockwise, true);
        // Mirrored both ways: bottom right.
        compare(b.br.clockwise, false);
        for (var id in b) compare(b[id].radius, 16)
    }

    function test_blocksOnEveryEdgeFollowTheReservedSpace() {
        var edges = ["top", "bottom", "left", "right"];
        for (var i = 0; i < edges.length; i++) {
            var s = resolve({
                "style": "frame",
                "edge": edges[i]
            }).sizes;
            var blocks = Geometry.cornerBlocks(s);
            compare(blocks.length, 4, edges[i]);
            blocks.forEach(function(b) {
                var a = b.atLeft ? s.reserved.left : s.reserved.right;
                var e = b.atTop ? s.reserved.top : s.reserved.bottom;
                compare(b.width, a + s.frameRadius, edges[i] + " " + b.id);
                compare(b.height, e + s.frameRadius, edges[i] + " " + b.id);
            });
        }
    }

    function test_aRadiusOfZeroGivesPlainRectanglesThatStillCoverTheCorners() {
        var b = blocks({
            "style": "frame",
            "frameRadius": 0
        });
        compare(Object.keys(b).length, 4, "the borders and the strip still meet at every corner");
        compare(b.tl.width, 8);
        compare(b.tl.height, 37, "the strip is at the top");
        compare(b.bl.height, 8);
        compare(b.tl.radius, 0);
    }

    function test_aBorderOfZeroKeepsTheRoundedCorners() {
        var b = blocks({
            "style": "frame",
            "frameBorder": 0
        });
        compare(Object.keys(b).length, 4);
        compare(b.bl.width, 16, "no left border, only the radius");
        compare(b.bl.height, 16);
        compare(b.tl.height, 37 + 16, "the strip is at the top");
    }

    function test_noBlockWhereThereIsNothingToDraw() {
        // Radius 0 and no border: only the corners next to the strip have a size, and only along the strip.
        var b = blocks({
            "style": "frame",
            "frameBorder": 0,
            "frameRadius": 0
        });
        compare(Object.keys(b).length, 0, "a strip with bare screen edges has no corner to fill");
    }

    // ---- fonts ----
    function test_genericFamiliesNeedNoCheck() {
        verify(Geometry.familyInstalled("sans-serif", []));
        verify(Geometry.familyInstalled("Monospace", []));
    }

    function test_installedFamiliesAreComparedWithoutCase() {
        verify(Geometry.familyInstalled("adwaita sans", ["Adwaita Sans", "FreeMono"]));
        verify(!Geometry.familyInstalled("No Such Font", ["Adwaita Sans"]));
    }

    function test_weightNames() {
        compare(Geometry.weightValue("bold"), 700);
        compare(Geometry.weightValue("thin"), 100);
        compare(Geometry.weightValue("regular"), 400);
        compare(Geometry.weightValue(null), 400);
        compare(Geometry.weightValue("nonsense"), 400);
    }

    name: "BarGeometry"
}
