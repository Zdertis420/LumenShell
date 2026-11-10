import QtQuick

// The source Prism creates. `api` is this module's own interface object; `slot` is what Prism
// tells the source (docs/prism.md#sources): the mode and the query the user typed, whether the
// window is open, and what the source may do: set its rows, show one line of message, and ask
// Prism to close. The item draws nothing.
Item {
    id: root

    required property var api
    property var slot
    readonly property int count: api.settings.values.rows

    // A row is {id, title, subtitle?, hint?, icon?, marks?}. Rows are replaced as a whole.
    // `marks` (version 1.1 of Prism) are [start, end) ranges in characters of the title, drawn
    // in bold and in the match color: here, the text the user typed.
    function build() {
        if (!slot)
            return ;

        if (slot.mode !== "default") {
            // Another mode: show what this source received, to see the prefix at work.
            slot.setRows([{
                "id": "mode",
                "title": "mode: " + slot.mode,
                "subtitle": "query: " + slot.query,
                "hint": "Echo"
            }]);
            // One line of message (version 1.1 of Prism); it is cleared when the text changes.
            slot.setMessage("The bundled search module handles the " + slot.mode + " mode too", "info");
            return ;
        }
        if (slot.query === "") {
            slot.setRows([]);
            return ;
        }
        const rows = [];
        for (let i = 1; i <= root.count; i++) rows.push({
            "id": "echo-" + i,
            "title": slot.query + " (" + i + ")",
            "subtitle": "Row " + i + " of " + root.count,
            "hint": "Echo",
            "marks": [[0, slot.query.length]]
        })
        // The second argument keeps the selected row selected if its id is still there; use it
        // when the rows are updated for the same query (for example when a slow search answers).
        // It applies only once the user has moved the selection; before that the first row stays.
        slot.setRows(rows, true);
    }

    Connections {
        function onQueryChanged() {
            root.build();
        }

        function onModeChanged() {
            root.build();
        }

        // Tab (version 1.2 of Prism): a source that can complete the text answers with
        // slot.setText(newText, cursor). This example has nothing to complete.
        function onCompleteRequested() {
        }

        // The user chose a row of this source: close Prism. A real source would act here.
        function onChosen(id) {
            root.api.log.info("chose " + id);
            root.slot.close();
        }

        target: root.slot
    }

}
