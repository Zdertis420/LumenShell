.pragma library

// The bar's fitting rule (FR-015a of Feature III, specs/003-bar/research.md R10), now along
// either axis (specs/004-bar-customization-styles): when the items do not fit the length of a
// screen, drop items from the end of the right list first, then the end of the center list,
// then the end of the left list, one at a time, until the rest fit. "Length" is the width of
// a horizontal bar and the height of a vertical one. Pure functions over numbers: no QML
// types, so they can be unit tested without a shell.

var PLACES = ["left", "center", "right"];
// Items are dropped from the places in this order.
var DROP_ORDER = ["right", "center", "left"];

// opts is {itemGap, placeGap, placePadding}. A plain number means one gap for items and
// places and no place padding (the shape of Feature III's calls).
function normalize(opts) {
    if (typeof opts === "number") {
        return {
            "itemGap": opts,
            "placeGap": opts,
            "placePadding": 0
        };
    }
    var o = opts || {};
    return {
        "itemGap": o.itemGap || 0,
        "placeGap": o.placeGap === undefined ? (o.itemGap || 0) : o.placeGap,
        "placePadding": o.placePadding || 0
    };
}

// The length of the first `count` items of a place, counting only the ones that take room
// (a length of 0 or less means the item is not shown).
function regionLength(lengths, count) {
    var shown = 0;
    var sum = 0;
    for (var i = 0; i < count && i < lengths.length; i++) {
        if (lengths[i] > 0) {
            shown++;
            sum += lengths[i];
        }
    }
    return {
        "shown": shown,
        "sum": sum
    };
}

// The length of one place for the given counts: its items, the item gaps between them, and
// the place padding at both ends. 0 when it holds nothing.
function widthOf(lengths, counts, place, opts) {
    var o = normalize(opts);
    var r = regionLength(lengths[place] || [], counts[place]);
    return r.shown === 0 ? 0 : r.sum + o.itemGap * (r.shown - 1) + 2 * o.placePadding;
}

// The total length for the given counts: the three places plus a place gap between every
// two places that hold something.
function totalWidth(lengths, counts, opts) {
    var o = normalize(opts);
    var places = 0;
    var total = 0;
    for (var p = 0; p < PLACES.length; p++) {
        var w = widthOf(lengths, counts, PLACES[p], o);
        if (w === 0)
            continue;
        places++;
        total += w;
    }
    return total + (places > 1 ? o.placeGap * (places - 1) : 0);
}

// fit(lengths, available, opts):
//   lengths:   {left: [n, ...], center: [...], right: [...]}, the measured length of each item
//              in list order (0 or less: not shown, takes no room)
//   available: the length of the bar
//   opts:      {itemGap, placeGap, placePadding} (or one number, see normalize)
// Returns {left, center, right, dropped, centerX}:
//   left, center, right: how many items of each list stay (the first n)
//   dropped:  [{place, index}] in the order they were dropped
//   centerX:  the start of the center place: centered on the bar when it fits there, otherwise
//             between the other two places
function fit(lengths, available, opts) {
    var o = normalize(opts);
    var counts = {
        "left": (lengths.left || []).length,
        "center": (lengths.center || []).length,
        "right": (lengths.right || []).length
    };
    var dropped = [];

    while (totalWidth(lengths, counts, o) > available) {
        var progressed = false;
        for (var d = 0; d < DROP_ORDER.length; d++) {
            var place = DROP_ORDER[d];
            if (counts[place] > 0) {
                counts[place]--;
                var index = counts[place];
                // An item that takes no room is not worth reporting.
                if ((lengths[place] || [])[index] > 0)
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

    var leftW = widthOf(lengths, counts, "left", o);
    var centerW = widthOf(lengths, counts, "center", o);
    var rightW = widthOf(lengths, counts, "right", o);
    var centerX = 0;
    if (centerW > 0) {
        var centered = (available - centerW) / 2;
        var leftEdge = leftW > 0 ? leftW + o.placeGap : 0;
        var rightEdge = available - (rightW > 0 ? rightW + o.placeGap : 0);
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
