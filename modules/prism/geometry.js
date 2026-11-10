.pragma library

// Prism's sizes and placement (specs/005-prism-window, contracts/prism-config.md and
// contracts/prism-surface.md). Pure functions: no QML types, so they are unit tested without
// a shell (scripts/tests/prism_geometry_test.qml).
//
// Input:
//   settings: the module's setting values. An unset size is null or undefined.
//   tokens:   {spacing: {xs, sm, md, lg, xl}, radius: {sm, md, lg}, font: {family, caption,
//             body, title}, border: {thin}}
//   area:     {width, height}: the size of the surface, which is the usable area of the screen
// Output of resolve: {sizes, warnings}; warnings are [{key, problem}], one per limited value.

var HALF = 0.5; // a radius is useful up to half of the smaller side of its shape
var TWICE = 2; // a margin or a padding is on both sides
var FIFTH = 5; // the default top placement is one fifth of the usable height (like Spotlight)
var CENTER_DIVISOR = 2;
var AA_RATIO = 4.5; // WCAG AA contrast for text
var LUMINANCE_OFFSET = 0.05; // the constant of the WCAG contrast ratio

function isUnset(v) {
    return v === null || v === undefined;
}

function orDefault(v, fallback) {
    return isUnset(v) ? fallback : v;
}

function clamp(v, lo, hi) {
    return Math.max(lo, Math.min(hi, v));
}

function warn(warnings, key, problem) {
    warnings.push({
        "key": key,
        "problem": problem
    });
}

var ANCHORS = ["top-left", "top", "top-right", "left", "center", "right", "bottom-left", "bottom", "bottom-right"];

// The row of an anchor ("top", "middle" or "bottom") and its column ("left", "center" or "right").
function rowOf(anchor) {
    if (anchor.indexOf("top") === 0)
        return "top";
    if (anchor.indexOf("bottom") === 0)
        return "bottom";
    return "middle";
}

function columnOf(anchor) {
    if (anchor === "top-left" || anchor === "left" || anchor === "bottom-left")
        return "left";
    if (anchor === "top-right" || anchor === "right" || anchor === "bottom-right")
        return "right";
    return "center";
}

// Resolves every size of the window for a surface of size `area`.
function resolve(settings, tokens, area) {
    var s = settings || {};
    var warnings = [];
    var anchor = ANCHORS.indexOf(s.anchor) >= 0 ? s.anchor : "top";
    var sp = tokens.spacing;
    var rd = tokens.radius;
    var margin = orDefault(s.margin, sp.lg);

    // The window may take at most the area minus the margin on both sides.
    var availW = Math.max(0, area.width - TWICE * margin);
    var availH = Math.max(0, area.height - TWICE * margin);
    if (TWICE * margin > Math.min(area.width, area.height)) {
        var m2 = Math.floor(Math.min(area.width, area.height) / TWICE / TWICE);
        warn(warnings, "margin", margin + " does not fit the screen; using " + m2);
        margin = m2;
        availW = Math.max(0, area.width - TWICE * margin);
        availH = Math.max(0, area.height - TWICE * margin);
    }

    var widthKey = isUnset(s.widthShare) ? "width" : "widthShare";
    var width = isUnset(s.widthShare) ? orDefault(s.width, 0) : Math.round(area.width * s.widthShare);
    if (width > availW) {
        warn(warnings, widthKey, "the width " + width + " does not fit the screen; using " + availW);
        width = availW;
    }
    var heightKey = isUnset(s.maxHeightShare) ? "maxHeight" : "maxHeightShare";
    var maxHeight = isUnset(s.maxHeightShare) ? orDefault(s.maxHeight, 0) : Math.round(area.height * s.maxHeightShare);
    if (maxHeight > availH) {
        warn(warnings, heightKey, "the height " + maxHeight + " does not fit the screen; using " + availH);
        maxHeight = availH;
    }

    var offsetX = orDefault(s.offsetX, 0);
    var offsetY = s.offsetY;
    if (isUnset(offsetY))
        offsetY = anchor === "top" ? Math.max(0, Math.floor(area.height / FIFTH) - margin) : 0;

    var base = orDefault(s.fontSize, tokens.font.body);
    // Room for the title and the subtitle (the body and caption text sizes) and the padding.
    var rowHeight = orDefault(s.rowHeight, base + tokens.font.caption + TWICE * sp.md);
    var iconSize = Math.max(1, orDefault(s.iconSize, rowHeight - TWICE * sp.sm));
    if (iconSize > rowHeight) {
        warn(warnings, "iconSize", "the icon " + iconSize + " is taller than the row " + rowHeight + "; using " + rowHeight);
        iconSize = rowHeight;
    }

    var radius = orDefault(s.radius, rd.lg);
    var smaller = Math.min(width, maxHeight);
    var radiusLimit = Math.max(0, Math.floor(smaller * HALF));
    if (radius > radiusLimit) {
        warn(warnings, "radius", radius + " is more than half of the smaller side of the window; using " + radiusLimit);
        radius = radiusLimit;
    }
    var rowRadius = orDefault(s.rowRadius, rd.md);
    var inputRadius = orDefault(s.inputRadius, rd.md);
    var rowLimit = Math.max(0, Math.floor(rowHeight * HALF));
    if (rowRadius > rowLimit) {
        warn(warnings, "rowRadius", rowRadius + " is more than half of the row height; using " + rowLimit);
        rowRadius = rowLimit;
    }
    if (inputRadius > rowLimit) {
        warn(warnings, "inputRadius", inputRadius + " is more than half of the input height; using " + rowLimit);
        inputRadius = rowLimit;
    }

    var sizes = {
        "anchor": anchor,
        "area": {
            "width": area.width,
            "height": area.height
        },
        "margin": margin,
        "width": width,
        "maxHeight": maxHeight,
        "offsetX": offsetX,
        "offsetY": offsetY,
        "maxRows": orDefault(s.maxRows, 1),
        "rowHeight": rowHeight,
        "iconSize": iconSize,
        "radius": radius,
        "inputRadius": inputRadius,
        "rowRadius": rowRadius,
        "borderWidth": orDefault(s.borderWidth, tokens.border.thin),
        "padding": orDefault(s.padding, sp.lg),
        "inputPadding": orDefault(s.inputPadding, sp.md),
        "rowPadding": orDefault(s.rowPadding, sp.md),
        "gap": orDefault(s.gap, sp.sm),
        "rowGap": orDefault(s.rowGap, sp.xs),
        "separatorWidth": orDefault(s.separatorWidth, tokens.border.thin),
        "shadowBlur": orDefault(s.shadowBlur, sp.xl),
        "shadowOffset": orDefault(s.shadowOffset, sp.sm),
        "slideDistance": sp.xl
    };

    // An offset that would push the window out of the area is reduced; say so once.
    var full = place(sizes, maxHeight);
    if (full.movedX)
        warn(warnings, "offsetX", "the offset " + offsetX + " would put the window outside the screen; the window was moved back");
    if (full.movedY)
        warn(warnings, "offsetY", "the offset " + offsetY + " would put the window outside the screen; the window was moved back");
    return {
        "sizes": sizes,
        "warnings": warnings
    };
}

// Where the panel goes for a content height `contentHeight` (the input, the gaps, the rows
// shown, the footer, the paddings), by the table of contracts/prism-surface.md. The offsets
// are in screen directions: positive X is right and positive Y is down. Returns
// {x, y, width, height, movedX, movedY}; moved* is true when the offset had to be reduced.
function place(sizes, contentHeight) {
    var a = sizes.area;
    var m = sizes.margin;
    var w = sizes.width;
    var h = clamp(contentHeight, 0, sizes.maxHeight);
    var col = columnOf(sizes.anchor);
    var row = rowOf(sizes.anchor);

    var x;
    if (col === "left")
        x = m + sizes.offsetX;
    else if (col === "right")
        x = a.width - m - w + sizes.offsetX;
    else
        x = Math.floor((a.width - w) / CENTER_DIVISOR) + sizes.offsetX;

    var y;
    if (row === "top")
        y = m + sizes.offsetY;
    else if (row === "bottom")
        y = a.height - m - h + sizes.offsetY;
    else
        y = Math.floor((a.height - sizes.maxHeight) / CENTER_DIVISOR) + sizes.offsetY; // the top stays put as the content changes

    var minX = m;
    var maxX = Math.max(m, a.width - m - w);
    var minY = m;
    var maxY = Math.max(m, a.height - m - h);
    // In the middle row the window is judged at its largest height, so it does not jump.
    if (row === "middle")
        maxY = Math.max(m, a.height - m - sizes.maxHeight);
    var cx = clamp(x, minX, maxX);
    var cy = clamp(y, minY, maxY);
    return {
        "x": cx,
        "y": cy,
        "width": w,
        "height": h,
        "movedX": cx !== x,
        "movedY": cy !== y
    };
}

// ---- fonts ----

// The families that need no check (CSS-style generic names, which the toolkit resolves).
var GENERIC_FAMILIES = ["sans-serif", "serif", "monospace", "cursive", "fantasy", "system-ui"];

// Whether a font family counts as installed: a generic name, or in `installed` (the list of
// family names from the toolkit), compared without regard to case.
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

var ROLES = ["input", "placeholder", "title", "subtitle", "hint", "footer", "empty"];

// The font of each text role: family, pixelSize, weight (a number) and italic. A role key
// that is unset follows the global key; the global key falls back to the tokens. The size
// of a role follows the global size when that is set and its own token otherwise (input: the
// title size, placeholder: the input's, title and empty: body, the others: caption).
// A family that is not installed falls back to the token family with one warning per key.
function fonts(settings, tokens, installed) {
    var s = settings || {};
    var warnings = [];
    var tokenSize = {
        "input": tokens.font.title,
        "title": tokens.font.body,
        "empty": tokens.font.body,
        "subtitle": tokens.font.caption,
        "hint": tokens.font.caption,
        "footer": tokens.font.caption
    };
    var globalFamily = tokens.font.family;
    if (!isUnset(s.fontFamily)) {
        if (familyInstalled(s.fontFamily, installed))
            globalFamily = s.fontFamily;
        else
            warn(warnings, "fontFamily", "\"" + s.fontFamily + "\" is not installed; using " + tokens.font.family);
    }
    var globalWeight = isUnset(s.fontWeight) ? "regular" : s.fontWeight;
    var globalItalic = s.fontItalic === true;
    var out = {
        "base": {
            "family": globalFamily,
            "pixelSize": orDefault(s.fontSize, tokens.font.body),
            "weight": weightValue(globalWeight),
            "italic": globalItalic
        }
    };
    for (var i = 0; i < ROLES.length; i++) {
        var role = ROLES[i];
        var family = globalFamily;
        var familyKey = role + "FontFamily";
        if (!isUnset(s[familyKey])) {
            if (familyInstalled(s[familyKey], installed))
                family = s[familyKey];
            else
                warn(warnings, familyKey, "\"" + s[familyKey] + "\" is not installed; using " + globalFamily);
        }
        var size;
        if (!isUnset(s[role + "FontSize"]))
            size = s[role + "FontSize"];
        else if (role === "placeholder")
            size = orDefault(out.input ? out.input.pixelSize : null, tokenSize.input);
        else if (!isUnset(s.fontSize))
            size = s.fontSize;
        else
            size = tokenSize[role];
        var weightName = isUnset(s[role + "FontWeight"]) ? globalWeight : s[role + "FontWeight"];
        var italic = isUnset(s[role + "FontItalic"]) ? globalItalic : s[role + "FontItalic"] === true;
        out[role] = {
            "family": family,
            "pixelSize": size,
            "weight": weightValue(weightName),
            "italic": italic
        };
    }
    return {
        "fonts": out,
        "warnings": warnings
    };
}

// ---- contrast (research.md R12) ----

// The part of a color that WCAG's relative luminance uses, for one channel in 0..1.
function channel(c) {
    var v = clamp(c, 0, 1);
    return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
}

function luminance(c) {
    return 0.2126 * channel(c.r) + 0.7152 * channel(c.g) + 0.0722 * channel(c.b);
}

// The WCAG contrast ratio of two colors ({r, g, b} in 0..1), from 1 to 21.
function contrast(a, b) {
    var la = luminance(a);
    var lb = luminance(b);
    var hi = Math.max(la, lb);
    var lo = Math.min(la, lb);
    return (hi + LUMINANCE_OFFSET) / (lo + LUMINANCE_OFFSET);
}

// `fg` over `bg` with the opacity `alpha` (0..1): the color that is seen.
function blend(fg, bg, alpha) {
    var k = clamp(alpha, 0, 1);
    return {
        "r": fg.r * k + bg.r * (1 - k),
        "g": fg.g * k + bg.g * (1 - k),
        "b": fg.b * k + bg.b * (1 - k)
    };
}

// The pairs that must reach AA (data-model.md "Contrast pairs"): [text key, background key].
var PAIRS = [["inputTextRole", "inputRole"], ["placeholderRole", "inputRole"], ["titleRole", "windowRole"], ["subtitleRole", "windowRole"], ["hintRole", "windowRole"], ["selectedTextRole", "selectedRole"], ["footerRole", "windowRole"], ["emptyRole", "windowRole"], ["matchRole", "windowRole"], ["messageRole", "windowRole"], ["errorRole", "windowRole"]];

// Checks the pairs. `c` has the base color of every role key of PAIRS ({r, g, b}) and the
// opacities `windowOpacity`, `inputOpacity` and `selectedOpacity`. The window background is
// taken as it is (what shows through it is not known); the input and the selected row are
// blended over it. Returns [{key, problem}], one per pair below 4.5.
function contrastWarnings(c) {
    var windowBg = c.windowRole;
    var warnings = [];
    for (var i = 0; i < PAIRS.length; i++) {
        var textKey = PAIRS[i][0];
        var bgKey = PAIRS[i][1];
        var bg = c[bgKey];
        if (c[textKey] === undefined || bg === undefined)
            continue;
        if (bgKey === "inputRole")
            bg = blend(c.inputRole, windowBg, orDefault(c.inputOpacity, 1));
        else if (bgKey === "selectedRole")
            bg = blend(c.selectedRole, windowBg, orDefault(c.selectedOpacity, 1));
        var ratio = contrast(c[textKey], bg);
        if (ratio < AA_RATIO)
            warn(warnings, textKey, "contrast against " + bgKey + " is " + ratio.toFixed(1) + ":1; text needs at least 4.5:1");
    }
    return warnings;
}
