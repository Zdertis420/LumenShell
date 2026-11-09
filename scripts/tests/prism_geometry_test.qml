import "../../modules/prism/geometry.js" as Geometry
import QtQuick
import QtTest

// Unit tests of Prism's sizes and placement (FR-016 to FR-020, FR-022, SC-002):
// modules/prism/geometry.js, run by qmltestrunner through `make test`.
TestCase {
    // ---- fonts ----
    // ---- contrast ----

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
            "family": "sans-serif",
            "caption": 11,
            "body": 13,
            "title": 16
        },
        "border": {
            "thin": 1
        }
    })
    readonly property var area: ({
        "width": 1000,
        "height": 800
    })
    readonly property var black: ({
        "r": 0,
        "g": 0,
        "b": 0
    })
    readonly property var white: ({
        "r": 1,
        "g": 1,
        "b": 1
    })

    function base(extra) {
        const s = {
            "anchor": "top",
            "offsetX": 0,
            "offsetY": 0,
            "margin": 16,
            "width": 640,
            "maxHeight": 480,
            "maxRows": 8
        };
        for (const k in extra) s[k] = extra[k]
        return s;
    }

    function placed(extra, h) {
        const r = Geometry.resolve(base(extra), tokens, area);
        return Geometry.place(r.sizes, h);
    }

    function test_nineAnchorsWithoutOffsets() {
        const want = {
            "top-left": [16, 16],
            "top": [180, 16],
            "top-right": [344, 16],
            "left": [16, 160],
            "center": [180, 160],
            "right": [344, 160],
            "bottom-left": [16, 584],
            "bottom": [180, 584],
            "bottom-right": [344, 584]
        };
        for (const name in want) {
            const p = placed({
                "anchor": name
            }, 200);
            compare(p.x, want[name][0], name + " x");
            compare(p.y, want[name][1], name + " y");
            compare(p.width, 640, name + " width");
            compare(p.height, 200, name + " height");
            verify(!p.movedX && !p.movedY, name + " moved");
        }
    }

    function test_defaultIsAFifthOfTheHeight() {
        const s = base({
        });
        delete s.offsetY;
        const r = Geometry.resolve(s, tokens, {
            "width": 1920,
            "height": 1040
        });
        const p = Geometry.place(r.sizes, 300);
        compare(p.y, 208);
        compare(p.x, 640);
        compare(r.warnings.length, 0);
    }

    function test_unsetOffsetIsZeroForOtherAnchors() {
        const s = base({
            "anchor": "bottom"
        });
        delete s.offsetY;
        compare(Geometry.resolve(s, tokens, area).sizes.offsetY, 0);
    }

    function test_offsetsAreScreenDirections() {
        const right = placed({
            "anchor": "right",
            "offsetX": -10
        }, 100);
        compare(right.x, 334);
        const left = placed({
            "anchor": "left",
            "offsetX": 10
        }, 100);
        compare(left.x, 26);
        const top = placed({
            "anchor": "top",
            "offsetY": 20
        }, 100);
        compare(top.y, 36);
        const bottom = placed({
            "anchor": "bottom",
            "offsetY": -20
        }, 100);
        compare(bottom.y, 800 - 16 - 100 - 20);
    }

    function test_theBottomEdgeStaysFixedAsTheContentGrows() {
        const small = placed({
            "anchor": "bottom"
        }, 100);
        const big = placed({
            "anchor": "bottom"
        }, 300);
        compare(small.y + small.height, big.y + big.height);
    }

    function test_theTopStaysFixedForTopAndMiddleRows() {
        for (const name of ["top", "left", "center", "right"]) {
            const a = placed({
                "anchor": name
            }, 100);
            const b = placed({
                "anchor": name
            }, 400);
            compare(a.y, b.y, name);
        }
    }

    function test_heightIsLimitedToTheLargestHeight() {
        compare(placed({
        }, 5000).height, 480);
        compare(placed({
        }, -5).height, 0);
    }

    function test_sizeLargerThanTheAreaIsLimitedWithAWarning() {
        const r = Geometry.resolve(base({
            "width": 5000,
            "maxHeight": 6000
        }), tokens, area);
        compare(r.sizes.width, 968);
        compare(r.sizes.maxHeight, 768);
        compare(r.warnings.length, 2);
        compare(r.warnings[0].key, "width");
        compare(r.warnings[1].key, "maxHeight");
    }

    function test_sharesReplaceTheSizes() {
        const r = Geometry.resolve(base({
            "widthShare": 0.5,
            "maxHeightShare": 0.25
        }), tokens, area);
        compare(r.sizes.width, 500);
        compare(r.sizes.maxHeight, 200);
        compare(r.warnings.length, 0);
    }

    function test_aShareTooLargeNamesTheShareKey() {
        const r = Geometry.resolve(base({
            "widthShare": 1
        }), tokens, area);
        compare(r.sizes.width, 968);
        compare(r.warnings[0].key, "widthShare");
    }

    function test_anOffsetThatPushesTheWindowOutIsMovedBackWithAWarning() {
        const r = Geometry.resolve(base({
            "anchor": "top-left",
            "offsetX": 5000,
            "offsetY": 5000
        }), tokens, area);
        const p = Geometry.place(r.sizes, 480);
        compare(p.x, 344);
        compare(p.y, 800 - 16 - 480);
        verify(p.movedX && p.movedY);
        compare(r.warnings.length, 2);
        compare(r.warnings[0].key, "offsetX");
        compare(r.warnings[1].key, "offsetY");
    }

    function test_theWindowIsAlwaysInsideTheArea() {
        for (const name of Geometry.ANCHORS) {
            for (const off of [-4096, -300, 0, 300, 4096]) {
                const p = placed({
                    "anchor": name,
                    "offsetX": off,
                    "offsetY": off
                }, 400);
                verify(p.x >= 16 && p.y >= 16, name + " " + off);
                verify(p.x + p.width <= 1000 - 16 && p.y + p.height <= 800 - 16, name + " " + off);
            }
        }
    }

    function test_unknownAnchorFallsBackToTop() {
        compare(Geometry.resolve(base({
            "anchor": "middle"
        }), tokens, area).sizes.anchor, "top");
    }

    function test_aMarginTooLargeIsLimited() {
        const r = Geometry.resolve(base({
            "margin": 500
        }), tokens, {
            "width": 400,
            "height": 300
        });
        verify(r.sizes.margin <= 75);
        compare(r.warnings[0].key, "margin");
    }

    function test_aTinyAreaNeverGivesNegativeSizes() {
        const r = Geometry.resolve(base({
        }), tokens, {
            "width": 10,
            "height": 10
        });
        verify(r.sizes.width >= 0 && r.sizes.maxHeight >= 0 && r.sizes.margin >= 0);
        const p = Geometry.place(r.sizes, 50);
        verify(p.width >= 0 && p.height >= 0);
    }

    function test_unsetSizesFollowTheTokens() {
        const r = Geometry.resolve({
            "anchor": "top",
            "width": 640,
            "maxHeight": 480,
            "maxRows": 8
        }, tokens, area);
        const s = r.sizes;
        compare(s.margin, 16);
        compare(s.padding, 16);
        compare(s.inputPadding, 12);
        compare(s.rowPadding, 12);
        compare(s.gap, 8);
        compare(s.rowGap, 4);
        compare(s.radius, 16);
        compare(s.rowRadius, 8);
        compare(s.inputRadius, 8);
        compare(s.borderWidth, 1);
        compare(s.separatorWidth, 1);
        compare(s.shadowBlur, 24);
        compare(s.shadowOffset, 8);
        compare(s.rowHeight, 13 + 11 + 24);
        compare(s.iconSize, 48 - 16);
        compare(r.warnings.length, 0);
    }

    function test_givenSizesWinOverTheTokens() {
        const r = Geometry.resolve(base({
            "radius": 4,
            "padding": 0,
            "rowHeight": 50,
            "iconSize": 20,
            "gap": 3
        }), tokens, area);
        compare(r.sizes.radius, 4);
        compare(r.sizes.padding, 0);
        compare(r.sizes.rowHeight, 50);
        compare(r.sizes.iconSize, 20);
        compare(r.sizes.gap, 3);
    }

    function test_aRadiusAboveHalfTheSmallerSideIsLimited() {
        const r = Geometry.resolve(base({
            "maxHeight": 100,
            "radius": 200,
            "rowHeight": 40,
            "rowRadius": 30,
            "inputRadius": 30
        }), tokens, area);
        compare(r.sizes.radius, 50);
        compare(r.sizes.rowRadius, 20);
        compare(r.sizes.inputRadius, 20);
        compare(r.warnings.length, 3);
    }

    function test_anIconTallerThanTheRowIsLimited() {
        const r = Geometry.resolve(base({
            "rowHeight": 20,
            "iconSize": 60
        }), tokens, area);
        compare(r.sizes.iconSize, 20);
        compare(r.warnings[0].key, "iconSize");
    }

    function test_fontsFollowTheTokensByDefault() {
        const f = Geometry.fonts({
        }, tokens, []).fonts;
        compare(f.base.family, "sans-serif");
        compare(f.base.pixelSize, 13);
        compare(f.input.pixelSize, 16);
        compare(f.placeholder.pixelSize, 16);
        compare(f.title.pixelSize, 13);
        compare(f.subtitle.pixelSize, 11);
        compare(f.hint.pixelSize, 11);
        compare(f.footer.pixelSize, 11);
        compare(f.empty.pixelSize, 13);
        compare(f.title.weight, 400);
        compare(f.title.italic, false);
    }

    function test_theGlobalFontReachesEveryRole() {
        const r = Geometry.fonts({
            "fontFamily": "Inter",
            "fontSize": 20,
            "fontWeight": "bold",
            "fontItalic": true
        }, tokens, ["Inter"]);
        compare(r.warnings.length, 0);
        for (const role of ["input", "placeholder", "title", "subtitle", "hint", "footer", "empty"]) {
            compare(r.fonts[role].family, "Inter", role);
            compare(r.fonts[role].pixelSize, 20, role);
            compare(r.fonts[role].weight, 700, role);
            compare(r.fonts[role].italic, true, role);
        }
    }

    function test_aRoleOverridesTheGlobalFont() {
        const f = Geometry.fonts({
            "fontSize": 20,
            "titleFontSize": 30,
            "titleFontWeight": "light",
            "titleFontItalic": false,
            "fontItalic": true
        }, tokens, []).fonts;
        compare(f.title.pixelSize, 30);
        compare(f.title.weight, 300);
        compare(f.title.italic, false);
        compare(f.subtitle.pixelSize, 20);
        compare(f.subtitle.italic, true);
    }

    function test_thePlaceholderFollowsTheInputSize() {
        const f = Geometry.fonts({
            "inputFontSize": 22
        }, tokens, []).fonts;
        compare(f.placeholder.pixelSize, 22);
        const g = Geometry.fonts({
            "inputFontSize": 22,
            "placeholderFontSize": 12
        }, tokens, []).fonts;
        compare(g.placeholder.pixelSize, 12);
    }

    function test_anUninstalledFamilyFallsBackWithOneWarningPerKey() {
        const r = Geometry.fonts({
            "fontFamily": "Nope Sans",
            "titleFontFamily": "Other Nope"
        }, tokens, ["Inter"]);
        compare(r.fonts.base.family, "sans-serif");
        compare(r.fonts.title.family, "sans-serif");
        compare(r.warnings.length, 2);
        compare(r.warnings[0].key, "fontFamily");
        compare(r.warnings[1].key, "titleFontFamily");
    }

    function test_genericAndCaseInsensitiveFamiliesAreInstalled() {
        verify(Geometry.familyInstalled("monospace", []));
        verify(Geometry.familyInstalled("inter", ["Inter"]));
        verify(!Geometry.familyInstalled("Nope", ["Inter"]));
    }

    function test_weightNames() {
        compare(Geometry.weightValue("thin"), 100);
        compare(Geometry.weightValue("black"), 900);
        compare(Geometry.weightValue("nonsense"), 400);
        compare(Geometry.weightValue(null), 400);
    }

    function test_contrastRatios() {
        fuzzyCompare(Geometry.contrast(black, white), 21, 0.01);
        fuzzyCompare(Geometry.contrast(white, white), 1, 0.001);
        fuzzyCompare(Geometry.contrast(white, black), Geometry.contrast(black, white), 0.0001);
    }

    function test_blend() {
        const b = Geometry.blend(white, black, 0.5);
        fuzzyCompare(b.r, 0.5, 0.0001);
        compare(Geometry.blend(white, black, 1).r, 1);
        compare(Geometry.blend(white, black, 0).r, 0);
        compare(Geometry.blend(white, black, 5).r, 1);
    }

    function colors(text, bg) {
        return {
            "windowRole": bg,
            "inputRole": bg,
            "selectedRole": bg,
            "inputTextRole": text,
            "placeholderRole": text,
            "titleRole": text,
            "subtitleRole": text,
            "hintRole": text,
            "selectedTextRole": text,
            "footerRole": text,
            "emptyRole": text,
            "inputOpacity": 1,
            "selectedOpacity": 1
        };
    }

    function test_goodContrastGivesNoWarnings() {
        compare(Geometry.contrastWarnings(colors(white, black)).length, 0);
    }

    function test_badContrastNamesBothKeys() {
        const w = Geometry.contrastWarnings(colors(white, white));
        compare(w.length, 8);
        verify(w[0].problem.indexOf("inputRole") >= 0);
        compare(w[0].key, "inputTextRole");
    }

    function test_opacityIsBlendedOverTheWindow() {
        const c = colors(white, black);
        c.inputRole = white;
        c.inputOpacity = 0;
        // The input background is invisible, so the text is judged against the black window.
        compare(Geometry.contrastWarnings(c).filter((w) => {
            return w.key === "inputTextRole";
        }).length, 0);
        c.inputOpacity = 1;
        compare(Geometry.contrastWarnings(c).filter((w) => {
            return w.key === "inputTextRole";
        }).length, 1);
    }

}
