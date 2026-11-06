# Module fixtures

Each subdirectory is a module used by the tests: valid ones, ones with a broken manifest (`err-*`, `dup-*`, `warn-*`), and ones with a broken or misbehaving entry file (`qml-*`, `helper-*`, `api-*`). Some of them contain deliberately broken QML, so this directory is excluded from `make fmt` and `make lint`.
