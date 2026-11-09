PROGRAMS := lumen config palette modules
BIN := scripts/bin
GO_SRC := $(wildcard scripts/src/*/*.go)
QML := $(shell find . -name '*.qml' -not -path './.claude/*' -not -path './.specify/*' -not -path './scripts/tests/fixtures/*')

.PHONY: build run fmt lint test qmltest smoke perf clean

build:
	@mkdir -p $(BIN)
	@for p in $(PROGRAMS); do \
		go build -o $(BIN)/$$p $$(ls scripts/src/$$p/*.go) || exit 1; \
	done

run: build
	$(BIN)/lumen start

fmt:
	gofmt -w scripts/src scripts/tests
	@test -z "$(QML)" || qmlformat -i $(QML)

lint:
	@out=$$(gofmt -l scripts/src scripts/tests); test -z "$$out" || { echo "gofmt needed:"; echo "$$out"; exit 1; }
	@for p in $(PROGRAMS); do \
		go vet $$(ls scripts/src/$$p/*.go) || exit 1; \
	done
	@test -z "$(QML)" || qmllint $(QML)

test: build
	go test $$(ls scripts/tests/*_test.go)
	@$(MAKE) --no-print-directory qmltest

# QML unit tests for the pure functions of the bar and of Prism (scripts/tests/bar_*_test.qml,
# scripts/tests/prism_*_test.qml).
# Skipped when there are none; fails with a clear message if the Qt 6 runner is missing.
# On some systems /usr/bin/qmltestrunner is the Qt 5 runner, which prints nothing and exits 1
# on Qt 6 tests, so the Qt 6 location is preferred.
QMLTEST := $(or $(wildcard /usr/lib/qt6/bin/qmltestrunner),$(shell command -v qmltestrunner6 2>/dev/null),qmltestrunner)

qmltest:
	@files=$$(ls scripts/tests/bar_*_test.qml scripts/tests/prism_*_test.qml 2>/dev/null); \
	if [ -z "$$files" ]; then exit 0; fi; \
	command -v $(QMLTEST) >/dev/null || { echo "qmltestrunner (Qt 6) not found"; exit 1; }; \
	for f in $$files; do QT_QPA_PLATFORM=offscreen $(QMLTEST) -input $$f || exit 1; done

smoke: build
	scripts/tests/smoke.sh $(if $(SECTION),--section $(SECTION))

# The long resource checks, run on their own (nothing else busy): memory over repeated actions
# and idle CPU. Prints the metric / budget / fact / previous-run table at the end.
perf: build
	scripts/tests/smoke.sh --section 15,15c --bar-cycles 100 --style-cycles 100 --cpu-seconds 30

clean:
	rm -rf $(BIN)
