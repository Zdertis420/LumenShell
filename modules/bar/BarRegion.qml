import QtQuick

// Holds the items of one place of the bar, in list order. It only positions its children;
// the bar creates the items as children of this row. (Not named Region: Quickshell has a
// type with that name, which would shadow this file.)
Row {
    required property string place
}
