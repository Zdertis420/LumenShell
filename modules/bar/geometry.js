.pragma library

// The bar's sizes (specs/004-bar-customization-styles, contracts/bar-surfaces.md and
// research.md R10): defaults for every unset size, per style, and the limits that a small
// screen puts on the rest. Pure functions: no QML types, so they are unit tested without a
// shell (scripts/tests/bar_geometry_test.qml).
//
// Input:
//   settings: the bar's setting values. An unset size is null or undefined.
//   tokens:   {spacing: {sm, md, lg, ...}, radius: {sm, md, lg, full}, font: {body}}
//   screen:   {width, height}, in logical pixels
// Output: {sizes, warnings}
//   sizes:    thickness, margin, padding, itemGap, placeGap, itemPadding, itemRadius, radius,
//             frameRadius, frameBorder, popupRadius, popupPadding, popupBorderWidth, fontSize,
//             vertical, reserved: {top, bottom, left, right}
//   warnings: [{key, problem}], one per value that was limited

var THICKNESS_MIN = 16; // the same minimum as the manifest rule of `thickness`
var HALF = 0.5; // the bar and frame may take at most this much of a screen axis
var TWICE = 2;

function isUnset(v) {
    return v === null || v === undefined;
}

function orDefault(v, fallback) {
    return isUnset(v) ? fallback : v;
}

function isVertical(edge) {
    return edge === "left" || edge === "right";
}

// The defaults of the sizes that depend on the style. The solid column is the bar of
// Feature III (SC-002).
function styleDefaults(style, tokens) {
    var sp = tokens.spacing;
    var rd = tokens.radius;
    if (style === "islands") {
        return {
            "margin": sp.sm,
            "padding": sp.md,
            "radius": rd.full
        };
    }
    return {
        "margin": 0,
        "padding": sp.sm,
        "radius": 0
    };
}

function warn(warnings, key, from, to, why) {
    warnings.push({
        "key": key,
        "problem": from + " does not fit the screen (" + why + "); using " + to
    });
}

function resolve(settings, tokens, screen) {
    var s = settings || {};
    var style = s.style === "islands" || s.style === "frame" ? s.style : "solid";
    var edge = s.edge === "bottom" || s.edge === "left" || s.edge === "right" ? s.edge : "top";
    var vertical = isVertical(edge);
    var frame = style === "frame";
    var def = styleDefaults(style, tokens);
    var warnings = [];

    var fontSize = orDefault(s.fontSize, tokens.font.body);
    var thickness = orDefault(s.thickness, fontSize + TWICE * tokens.spacing.md);
    var margin = orDefault(s.margin, def.margin);
    var frameBorder = frame ? orDefault(s.frameBorder, tokens.spacing.sm) : 0;

    // Along the direction across the bar (the height of a horizontal bar, the width of a
    // vertical one), the bar's own size, its margin and the frame border on the opposite side
    // may take at most half of the screen. Reduce thickness first, then margin, then border.
    var across = vertical ? screen.width : screen.height;
    var limitAcross = Math.floor(across * HALF);
    var over = thickness + margin + frameBorder - limitAcross;
    if (over > 0) {
        var cutT = Math.min(over, Math.max(0, thickness - THICKNESS_MIN));
        if (cutT > 0) {
            warn(warnings, "thickness", thickness, thickness - cutT, "the bar and frame may take at most half of the screen");
            thickness -= cutT;
            over -= cutT;
        }
        var cutM = Math.min(over, margin);
        if (cutM > 0) {
            warn(warnings, "margin", margin, margin - cutM, "the bar and frame may take at most half of the screen");
            margin -= cutM;
            over -= cutM;
        }
        var cutB = Math.min(over, frameBorder);
        if (cutB > 0) {
            warn(warnings, "frameBorder", frameBorder, frameBorder - cutB, "the bar and frame may take at most half of the screen");
            frameBorder -= cutB;
        }
    }
    // Along the bar the margins (and the frame's border on both ends) take the same share.
    var along = vertical ? screen.height : screen.width;
    var limitAlong = Math.floor(along * HALF);
    var overAlong = TWICE * (margin + frameBorder) - limitAlong;
    if (overAlong > 0) {
        var keepM = Math.max(0, margin - Math.ceil(overAlong / TWICE));
        if (keepM !== margin) {
            warn(warnings, "margin", margin, keepM, "the margins may take at most half of the screen along the bar");
            margin = keepM;
        }
        overAlong = TWICE * (margin + frameBorder) - limitAlong;
        if (overAlong > 0) {
            var keepB = Math.max(0, frameBorder - Math.ceil(overAlong / TWICE));
            if (keepB !== frameBorder) {
                warn(warnings, "frameBorder", frameBorder, keepB, "the frame may take at most half of the screen along the bar");
                frameBorder = keepB;
            }
        }
    }

    // What the bar and the frame give up on each side of the screen.
    var barSpace = thickness + margin;
    var reserved = {
        "top": 0,
        "bottom": 0,
        "left": 0,
        "right": 0
    };
    if (frame) {
        reserved.top = frameBorder;
        reserved.bottom = frameBorder;
        reserved.left = frameBorder;
        reserved.right = frameBorder;
    }
    reserved[edge] = barSpace;

    // The frame's inner corner radius: at most half of the smaller side of the area inside.
    var frameRadius = orDefault(s.frameRadius, tokens.radius.lg);
    if (frame) {
        var innerW = screen.width - reserved.left - reserved.right;
        var innerH = screen.height - reserved.top - reserved.bottom;
        var limitR = Math.max(0, Math.floor(Math.min(innerW, innerH) * HALF));
        if (frameRadius > limitR) {
            warn(warnings, "frameRadius", frameRadius, limitR, "at most half of the smaller side of the area inside the frame");
            frameRadius = limitR;
        }
    }

    // Radii are not limited: a larger radius is drawn fully round (FR-015), with no warning.
    return {
        "sizes": {
            "style": style,
            "edge": edge,
            "vertical": vertical,
            "fontSize": fontSize,
            "thickness": thickness,
            "margin": margin,
            "padding": orDefault(s.padding, def.padding),
            "itemGap": orDefault(s.itemGap, tokens.spacing.sm),
            "placeGap": orDefault(s.placeGap, tokens.spacing.sm),
            "itemPadding": orDefault(s.itemPadding, tokens.spacing.md),
            "itemRadius": orDefault(s.itemRadius, tokens.radius.sm),
            "radius": orDefault(s.radius, def.radius),
            "frameRadius": frameRadius,
            "frameBorder": frameBorder,
            "popupRadius": orDefault(s.popupRadius, tokens.radius.md),
            "popupPadding": orDefault(s.popupPadding, 0),
            "popupBorderWidth": orDefault(s.popupBorderWidth, 1),
            "reserved": reserved
        },
        "warnings": warnings
    };
}

// The corner blocks of the frame (contracts/bar-surfaces.md "Corner blocks", research R4a).
// The compositor lays a panel out inside the space that earlier panels reserved, so the panels
// can come up short at the screen corners. A corner block starts at the screen corner and
// covers the whole corner square: the space reserved on its side plus the radius across, the
// space reserved at its end plus the radius along. It is the frame color everywhere but a
// quarter disc of the radius (the rounded inner corner), whose center is the block's corner
// farthest from the screen corner. The block is drawn as two simple shapes that do not
// overlap, not as a rectangle with a hole (a hole that shares edges with the outline is
// rendered wrongly at one corner):
//   band:     the block without its inner square (an L shape), six points
//   spandrel: the inner square without the quarter disc: from p0 (the inner corner) along
//             the square's side to p1, around the arc to p2, and back to p0
// Returns one entry per corner that has a size:
// {id: "tl"|"tr"|"bl"|"br", atTop, atLeft, width, height, radius, band, p0, p1, p2,
// clockwise} in the block's own coordinates. Not the frame style: no blocks.
function cornerBlocks(sizes) {
    if (!sizes || sizes.style !== "frame")
        return [];
    var r = sizes.frameRadius;
    var res = sizes.reserved;
    var corners = [
        {"id": "tl", "atTop": true, "atLeft": true, "a": res.left, "b": res.top},
        {"id": "tr", "atTop": true, "atLeft": false, "a": res.right, "b": res.top},
        {"id": "bl", "atTop": false, "atLeft": true, "a": res.left, "b": res.bottom},
        {"id": "br", "atTop": false, "atLeft": false, "a": res.right, "b": res.bottom}
    ];
    var out = [];
    for (var i = 0; i < corners.length; i++) {
        var c = corners[i];
        var width = c.a + r;
        var height = c.b + r;
        if (width <= 0 || height <= 0)
            continue;
        // Work in a frame whose origin is the screen corner and whose axes point inward (u across
        // the side, v along the end), then map each point to the block's own coordinates.
        var mapPoint = function(u, v) {
            return {
                "x": c.atLeft ? u : width - u,
                "y": c.atTop ? v : height - v
            };
        };
        var a = c.a;
        var b = c.b;
        // Mirroring once reverses the arc: from (a + r, b) to (a, b + r) it is counterclockwise
        // in the inward frame, so it is clockwise when exactly one axis is mirrored.
        var mirrored = (c.atLeft ? 0 : 1) + (c.atTop ? 0 : 1);
        out.push({
            "id": c.id,
            "atTop": c.atTop,
            "atLeft": c.atLeft,
            "width": width,
            "height": height,
            "radius": r,
            "band": [mapPoint(0, 0), mapPoint(width, 0), mapPoint(width, b), mapPoint(a, b), mapPoint(a, height), mapPoint(0, height)],
            "p0": mapPoint(a, b),
            "p1": mapPoint(a + r, b),
            "p2": mapPoint(a, b + r),
            "clockwise": mirrored === 1
        });
    }
    return out;
}

// The most a radius can usefully be for a shape: half of its smaller side. A radius above it
// is drawn the same, so QML can pass any value through this.
function roundRadius(radius, width, height) {
    return Math.max(0, Math.min(radius, Math.floor(Math.min(width, height) * HALF)));
}

// The families that need no check (CSS-style generic names, which the toolkit resolves).
var GENERIC_FAMILIES = ["sans-serif", "serif", "monospace", "cursive", "fantasy", "system-ui"];

// Whether a font family counts as installed: it is a generic name, or it is in `installed`
// (the list of family names from the toolkit), compared without regard to case.
function familyInstalled(family, installed) {
    var wanted = String(family).toLowerCase();
    if (GENERIC_FAMILIES.indexOf(wanted) >= 0)
        return true;
    for (var i = 0; i < installed.length; i++) {
        if (String(installed[i]).toLowerCase() === wanted)
            return true;
    }
    return false;
}

// Qt font weights (the numeric values of Font.Thin to Font.Black) by the setting's names.
var WEIGHTS = {
    "thin": 100,
    "light": 300,
    "regular": 400,
    "medium": 500,
    "semibold": 600,
    "bold": 700,
    "black": 900
};

function weightValue(name) {
    return WEIGHTS[name] === undefined ? WEIGHTS.regular : WEIGHTS[name];
}
