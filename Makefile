PROGRAMS := lumen config palette modules
BIN := scripts/bin
GO_SRC := $(wildcard scripts/src/*/*.go)
QML := $(shell find . -name '*.qml' -not -path './.claude/*' -not -path './.specify/*' -not -path './scripts/tests/fixtures/*')

.PHONY: build run fmt lint test smoke clean

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

smoke: build
	scripts/tests/smoke.sh

clean:
	rm -rf $(BIN)
