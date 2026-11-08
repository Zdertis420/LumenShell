.pragma library

// The bar's fitting rule (FR-015a, specs/003-bar/research.md R10): when the items do not fit
// the width of a screen, drop items from the end of the right list first, then the end of
// the center list, then the end of the left list, one at a time, until the rest fit.
// Pure functions over numbers: no QML types, so they can be unit tested without a shell.

var PLACES = ["left", "center", "right"];
// Items are dropped from the places in this order.
var DROP_ORDER = ["right", "center", "left"];

// The width of the first `count` items of a place, counting only the ones that take room
// (a width of 0 or less means the item is not shown) and the gaps between those.
function regionWidth(widths, count) {
    var shown = 0;
    var sum = 0;
    for (var i = 0; i < count && i < widths.length; i++) {
        if (widths[i] > 0) {
            shown++;
            sum += widths[i];
        }
    }
    return {
        "shown": shown,
        "sum": sum
    };
}

// The total width of the bar for the given counts: the three regions plus a gap between
// every two items and between every two regions that hold something.
function totalWidth(widths, counts, gap) {
    var regions = 0;
    var total = 0;
    for (var p = 0; p < PLACES.length; p++) {
        var place = PLACES[p];
        var r = regionWidth(widths[place] || [], counts[place]);
        if (r.shown === 0)
            continue;
        regions++;
        total += r.sum + gap * (r.shown - 1);
    }
    return total + (regions > 1 ? gap * (regions - 1) : 0);
}

// The width of one region for the given counts (0 when it holds nothing).
function widthOf(widths, counts, place, gap) {
    var r = regionWidth(widths[place] || [], counts[place]);
    return r.shown === 0 ? 0 : r.sum + gap * (r.shown - 1);
}

// fit(widths, available, gap):
//   widths:    {left: [w, ...], center: [...], right: [...]}, the measured width of each item
//              in list order (0 or less: not shown, takes no room)
//   available: the width of the bar
//   gap:       the space between two items and between two regions
// Returns {left, center, right, dropped, centerX}:
//   left, center, right: how many items of each list stay (the first n)
//   dropped:  [{place, index}] in the order they were dropped
//   centerX:  the x of the center region: centered on the bar when it fits there, otherwise
//             between the other two regions
function fit(widths, available, gap) {
    var counts = {
        "left": (widths.left || []).length,
        "center": (widths.center || []).length,
        "right": (widths.right || []).length
    };
    var dropped = [];

    while (totalWidth(widths, counts, gap) > available) {
        var progressed = false;
        for (var d = 0; d < DROP_ORDER.length; d++) {
            var place = DROP_ORDER[d];
            if (counts[place] > 0) {
                counts[place]--;
                var index = counts[place];
                // An item that takes no room is not worth reporting.
                if ((widths[place] || [])[index] > 0)
                    dropped.push({
                        "place": place,
                        "index": index
                    });
                progressed = true;
                break;
            }
        }
        if (!progressed)
            break;
    }

    var leftW = widthOf(widths, counts, "left", gap);
    var centerW = widthOf(widths, counts, "center", gap);
    var rightW = widthOf(widths, counts, "right", gap);
    var centerX = 0;
    if (centerW > 0) {
        var centered = (available - centerW) / 2;
        var leftEdge = leftW > 0 ? leftW + gap : 0;
        var rightEdge = available - (rightW > 0 ? rightW + gap : 0);
        if (centered >= leftEdge && centered + centerW <= rightEdge) {
            centerX = centered;
        } else {
            centerX = leftEdge + Math.max(0, (rightEdge - leftEdge - centerW) / 2);
        }
    }
    return {
        "left": counts.left,
        "center": counts.center,
        "right": counts.right,
        "dropped": dropped,
        "centerX": centerX
    };
}
