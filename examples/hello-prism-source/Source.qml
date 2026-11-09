import QtQuick

// The source Prism creates. `api` is this module's own interface object; `slot` is what
// Prism tells the source (specs/005-prism-window/contracts/prism-source-slot.md): the mode
// and the query the user typed, whether the window is open, and two things the source may
// do, set its rows and ask Prism to close. The item draws nothing.
Item {
    id: root

    required property var api
    property var slot
    readonly property int count: api.settings.values.rows

    // A row is {id, title, subtitle?, hint?, icon?}. Rows are replaced as a whole.
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
            "hint": "Echo"
        })
        slot.setRows(rows);
    }

    Connections {
        function onQueryChanged() {
            root.build();
        }

        function onModeChanged() {
            root.build();
        }

        // The user chose a row of this source: close Prism. A real source would act here.
        function onChosen(id) {
            root.api.log.info("chose " + id);
            root.slot.close();
        }

        target: root.slot
    }

}
