import QtQuick
import Quickshell

// Reads or calls every documented api member once and logs the result. It also reports
// every change of its settings, setting errors, and the palette.
Scope {
    id: root

    required property var api

    readonly property var settingValues: api.settings.values
    readonly property var settingErrors: api.settings.errors

    onSettingValuesChanged: api.log.info("probe settings changed greeting=" + settingValues.greeting + " size=" + settingValues.size)
    onSettingErrorsChanged: api.log.info("probe setting errors " + settingErrors.map((e) => e.key + ":" + e.problem).join(";"))

    Component.onCompleted: {
        api.log.info("probe id " + api.id + " apiVersion " + api.apiVersion);
        api.log.info("probe theme background " + api.theme.background);
        api.log.info("probe style spacing.md " + api.style.spacing.md);
        api.log.info("probe settings.greeting " + api.settings.values.greeting);
        const mods = api.modules.list;
        api.log.info("probe modules.list " + mods.length + " manifest " + (mods.length > 0 && mods[0].manifest ? "yes" : "no"));
        api.log.info("probe log.recent " + api.log.recent().length);
        api.log.info("probe config.value theme.source " + api.config.value("theme.source"));
        api.log.info("probe paletteId " + api.theme.paletteId);
        api.run("probe", {}, 1500, req => api.log.info("probe run state " + req.state + " message " + (req.data ? req.data.message : "")));
    }

    Connections {
        target: root.api.theme

        function onPaletteIdChanged() {
            root.api.log.info("probe paletteId " + root.api.theme.paletteId);
        }
    }
}
