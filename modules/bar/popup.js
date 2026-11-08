.pragma library

// Where a popup goes (FR-038, specs/003-bar/contracts/bar-slot.md): next to the item that
// opened it, inside the screen, never over the item, and at most a fraction of the screen
// in each direction. Pure functions over numbers, so they can be unit tested without a shell.

var MIN_SIZE = 1;

function clamp(value, low, high) {
    return Math.min(Math.max(value, low), high);
}

// place(anchor, size, screen, edge, maxFraction)
//   anchor:      {x, y, width, height} of the item, in screen coordinates
//   size:        {width, height} the content wishes for
//   screen:      {x, y, width, height}
//   edge:        "top" (the popup goes below the item) or "bottom" (above it)
//   maxFraction: the most of the screen's width and height a popup may take (0 to 1)
// Returns {x, y, width, height}.
function place(anchor, size, screen, edge, maxFraction) {
    var maxW = Math.max(MIN_SIZE, Math.floor(screen.width * maxFraction));
    var maxH = Math.max(MIN_SIZE, Math.floor(screen.height * maxFraction));
    var width = clamp(Math.ceil(size.width), MIN_SIZE, maxW);
    var height = clamp(Math.ceil(size.height), MIN_SIZE, maxH);

    // Centered on the item, then moved to stay on the screen.
    var x = anchor.x + (anchor.width - width) / 2;
    x = clamp(x, screen.x, screen.x + screen.width - width);

    // Below the item for a top bar, above it for a bottom bar; the room on that side limits
    // the height, so the popup never reaches over the item.
    var y;
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
    return {
        "x": Math.round(x),
        "y": Math.round(y),
        "width": width,
        "height": height
    };
}
