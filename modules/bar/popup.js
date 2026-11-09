.pragma library

// Where a popup goes (FR-038, specs/003-bar/contracts/bar-slot.md; four edges since
// specs/004-bar-customization-styles): next to the item that opened it, on the side that
// faces into the screen, inside the screen, never over the item, and at most a fraction of
// the screen in each direction. Pure functions over numbers, so they can be unit tested
// without a shell.

var MIN_SIZE = 1;

function clamp(value, low, high) {
    return Math.min(Math.max(value, low), high);
}

// place(anchor, size, screen, edge, maxFraction)
//   anchor:      {x, y, width, height} of the item, in screen coordinates
//   size:        {width, height} the content wishes for
//   screen:      {x, y, width, height}
//   edge:        the bar's edge. "top": the popup goes below the item. "bottom": above it.
//                "left": right of the item. "right": left of it
//   maxFraction: the most of the screen's width and height a popup may take (0 to 1)
// Returns {x, y, width, height}.
function place(anchor, size, screen, edge, maxFraction) {
    var maxW = Math.max(MIN_SIZE, Math.floor(screen.width * maxFraction));
    var maxH = Math.max(MIN_SIZE, Math.floor(screen.height * maxFraction));
    var width = clamp(Math.ceil(size.width), MIN_SIZE, maxW);
    var height = clamp(Math.ceil(size.height), MIN_SIZE, maxH);
    var x;
    var y;

    if (edge === "left" || edge === "right") {
        // Centered on the item along the bar, then moved to stay on the screen.
        y = anchor.y + (anchor.height - height) / 2;
        y = clamp(y, screen.y, screen.y + screen.height - height);
        // Right of the item for a left bar, left of it for a right bar; the room on that side
        // limits the width, so the popup never reaches over the item.
        if (edge === "right") {
            var before = anchor.x - screen.x;
            width = Math.max(MIN_SIZE, Math.min(width, before));
            x = anchor.x - width;
        } else {
            var afterStart = anchor.x + anchor.width;
            var after = screen.x + screen.width - afterStart;
            width = Math.max(MIN_SIZE, Math.min(width, after));
            x = afterStart;
        }
    } else {
        // Centered on the item, then moved to stay on the screen.
        x = anchor.x + (anchor.width - width) / 2;
        x = clamp(x, screen.x, screen.x + screen.width - width);
        // Below the item for a top bar, above it for a bottom bar; the room on that side limits
        // the height, so the popup never reaches over the item.
        if (edge === "bottom") {
            var above = anchor.y - screen.y;
            height = Math.max(MIN_SIZE, Math.min(height, above));
            y = anchor.y - height;
        } else {
            var belowStart = anchor.y + anchor.height;
            var below = screen.y + screen.height - belowStart;
            height = Math.max(MIN_SIZE, Math.min(height, below));
            y = belowStart;
        }
    }
    return {
        "x": Math.round(x),
        "y": Math.round(y),
        "width": width,
        "height": height
    };
}
