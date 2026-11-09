# Contract: the bar's surfaces and geometry

What the bar puts on each screen for each style, how it reserves space and takes input, and how sizes are limited. The numbers of the default look are in [bar-config-1.2.md](bar-config-1.2.md).

## Surfaces

| Style | Surfaces | Reserved space | Input |
|-------|----------|----------------|-------|
| `solid` | the strip panel | `thickness + margin` on the bar's edge | the whole strip |
| `islands` | the strip panel (transparent; the islands are drawn inside it), mapped only while at least one item is shown | `thickness + margin` on the bar's edge, and nothing while no item is shown | the union of the islands only; the gaps let the pointer through |
| `frame` | the strip panel, up to 3 border panels, up to 4 corner blocks | strip `thickness + margin` on the bar's edge; `frameBorder` on each border side; the corner blocks reserve nothing | the strip panel; the borders and corner blocks take no input |

- An island is drawn only for a place that holds at least one shown item. In the islands style, when no place holds a shown item on a screen, the strip panel is not mapped: no surface, no reserved space (the panel object and its items stay, so the island and the space come back as soon as an item is shown, with no item recreated).
- There is one strip panel per shown screen, and it owns the items. A style or edge change re-draws and re-anchors it; items are not recreated by it (FR-005). If spike R3(b) shows that a live anchor change is not possible, the panel is recreated on an edge change and the items once with it.
- Border panels exist only in the frame style and only when `frameBorder` is above 0. Corner blocks exist only in the frame style, one per screen corner whose block has a size (see below), also when `frameRadius` is 0. They are created and released by lazy loaders.
- **Why corner blocks (fix of 2026-11-08).** The compositor lays a layer surface out inside the space that earlier surfaces in the same layer reserved, so a panel anchored along an edge loses the zone of each panel at its ends (the strip, the opposite border and the side borders each come up short by the other panels' widths). Nothing in the panels covers the screen corners then. The corner blocks do: they start at the screen corner, ignore the other surfaces' zones, and are drawn in the frame color, so the band is unbroken whatever order the panels are laid out in.

- **No margins on border panels (fix of 2026-11-09).** A border panel is anchored to its side (and, across the bar, to both ends of it) with **no margins**: the compositor already places it inside the space that earlier surfaces reserved, so a margin for the strip's space counts twice and leaves a gap of the strip's width next to the corner block. Any shortfall left at a screen end is at most one border width, which the corner block (`border + radius` wide) covers.

### Corner blocks

For a corner with `a` = the space reserved on its side (left or right) and `b` = the space reserved at its end (top or bottom), both from `reserved` in [data-model.md](../data-model.md), and `r` = `frameRadius`: the block is a rectangle of `a + r` by `b + r` anchored to the screen corner with no margin. It is filled with the frame color everywhere **except** the quarter disc of radius `r` whose center is the block's corner farthest from the screen corner (the transparent part of the rounded inner corner). It is drawn as two shapes that do not overlap: the L-shaped band (the block without its `r` by `r` inner square) and the spandrel (that square without the quarter disc). A rectangle with the disc cut out by a fill rule is not used, because the renderer fills the cut-out wrongly at one corner. With `r = 0` it is a plain rectangle `a` by `b`. A block whose width or height is 0 is not created.
- Everything is in the same layer as the bar, so a fullscreen window hides it as it hides the bar. Windows are never drawn under reserved space.
- A screen the `screens` setting excludes has no surface of the bar, frame included.

## Drawing

| Part | Shape | Color |
|------|-------|-------|
| Solid strip | rectangle with `radius`, inset by `margin` | `backgroundRole` with `backgroundOpacity` |
| Island | rounded rectangle per non-empty place, `radius`, with `padding` at its ends; empty places draw nothing | `islandRole` with `islandOpacity` |
| Frame | strip and border panels, rectangles | `frameRole` with `frameOpacity` (one color, no seam) |
| Frame corner block | the corner square of the frame, without the quarter disc of the rounded inner corner (see "Corner blocks") | same as the frame |
| Focus indicator | outline around the focused item, `itemRadius` | `focusRole` |

## Layout along the bar

1. Each item's length comes from the item (see [bar-slot-1.2.md](bar-slot-1.2.md)), limited to `maxLength`.
2. `layout.js` fits the three places into the length between the strip's end paddings (solid, frame) or into the island lengths (islands) with `itemGap` between items and `placeGap` between places, dropping items in the order of Feature III (end of right, then center, then left) until the rest fit. The length is the width on a horizontal bar and the height on a vertical one.
3. In the islands style each non-empty place adds `2 x padding` to its length.
4. The center place is centered on the bar when it fits there, otherwise between the other two places (as before).
5. On a vertical bar the left place is at the top, the center in the middle, the right at the bottom.

## Limits for small screens (`geometry.js`)

Applied per screen, each with one warning naming the key:

| Rule | Limit |
|------|-------|
| Space taken on one axis by the bar and the frame | at most half of the screen's length on that axis |
| `thickness`, `margin`, `frameBorder` | reduced to fit the rule above, in that order |
| `frameRadius` | at most half of the smaller side of the area inside the frame |
| `radius`, `itemRadius`, `popupRadius` | no limit and no warning: a larger value is drawn fully round |
| Text that does not fit the thickness | clipped; no warning (the user chose both) |

## Popup placement

`popup.js` places the popup relative to the item's rectangle on the screen:

| Edge | Popup side | Aligned by |
|------|-----------|------------|
| `top` | below the item | the item's horizontal center, shifted to stay on screen |
| `bottom` | above the item | same |
| `left` | right of the item | the item's vertical center, shifted to stay on screen |
| `right` | left of the item | same |

The size is limited to half of the screen on each axis and to the room on its side; it never overlaps the item.
