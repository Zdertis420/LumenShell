import "../../modules/prism/prefix.js" as Prefix
import QtQuick
import QtTest

// Unit tests of the mode prefix (specs/005-prism-window, FR-010 prefix): modules/prism/prefix.js,
// run by qmltestrunner through `make test`.
TestCase {
    readonly property var letters: Prefix.defaultLetters()

    function test_noPrefixIsDefaultWithTheWholeText() {
        const r = Prefix.parse("report", letters);
        compare(r.mode, "default");
        compare(r.query, "report");
        compare(r.prefixLength, 0);
    }

    function test_emptyAndNullText() {
        compare(Prefix.parse("", letters).mode, "default");
        compare(Prefix.parse("", letters).query, "");
        compare(Prefix.parse(null, letters).query, "");
        compare(Prefix.parse(undefined, letters).mode, "default");
    }

    function test_prefixWithAQuery() {
        const r = Prefix.parse("@f report", letters);
        compare(r.mode, "files");
        compare(r.query, "report");
        compare(r.prefixLength, 2);
    }

    function test_prefixAloneHasAnEmptyQuery() {
        const r = Prefix.parse("@f", letters);
        compare(r.mode, "files");
        compare(r.query, "");
        compare(r.prefixLength, 2);
    }

    function test_prefixWithTrailingSpaceHasAnEmptyQuery() {
        const r = Prefix.parse("@a ", letters);
        compare(r.mode, "apps");
        compare(r.query, "");
    }

    function test_onlyOneSpaceIsTakenOffTheQuery() {
        compare(Prefix.parse("@g  two", letters).query, " two");
    }

    function test_letterFollowedByTextIsNotAPrefix() {
        const r = Prefix.parse("@fx", letters);
        compare(r.mode, "default");
        compare(r.query, "@fx");
        compare(r.prefixLength, 0);
    }

    function test_unknownLetterIsPlainText() {
        const r = Prefix.parse("@x y", letters);
        compare(r.mode, "default");
        compare(r.query, "@x y");
    }

    function test_doubleSignIsPlainText() {
        compare(Prefix.parse("@@f", letters).mode, "default");
        compare(Prefix.parse("@@f x", letters).query, "@@f x");
    }

    function test_signAloneIsPlainText() {
        compare(Prefix.parse("@", letters).mode, "default");
        compare(Prefix.parse("@", letters).query, "@");
    }

    function test_prefixMustBeAtTheStart() {
        compare(Prefix.parse(" @f x", letters).mode, "default");
        compare(Prefix.parse("x @f", letters).mode, "default");
    }

    function test_lettersAreCaseSensitive() {
        compare(Prefix.parse("@F x", letters).mode, "default");
    }

    function test_allFiveDefaultModes() {
        compare(Prefix.parse("@a x", letters).mode, "apps");
        compare(Prefix.parse("@f x", letters).mode, "files");
        compare(Prefix.parse("@w x", letters).mode, "web");
        compare(Prefix.parse("@s x", letters).mode, "commands");
        compare(Prefix.parse("@g x", letters).mode, "grep");
    }

    function test_changedLettersChangeTheMatch() {
        const mine = {
            "apps": "p",
            "files": "f",
            "web": "w",
            "commands": "s",
            "grep": "g"
        };
        compare(Prefix.parse("@p x", mine).mode, "apps");
        compare(Prefix.parse("@a x", mine).mode, "default");
    }

    function test_swappedLettersAreValid() {
        const r = Prefix.checkLetters({
            "prefixApps": "f",
            "prefixFiles": "a"
        });
        compare(r.errors.length, 0);
        compare(r.letters.apps, "f");
        compare(r.letters.files, "a");
        compare(Prefix.parse("@f x", r.letters).mode, "apps");
    }

    function test_defaultsGiveNoErrors() {
        const r = Prefix.checkLetters({
        });
        compare(r.errors.length, 0);
        compare(r.letters.grep, "g");
    }

    function test_duplicateFallsBackToTheDefaultAndNamesTheKey() {
        const r = Prefix.checkLetters({
            "prefixWeb": "a"
        });
        compare(r.errors.length, 1);
        compare(r.errors[0].key, "prefixWeb");
        compare(r.errors[0].fallback, "w");
        compare(r.letters.web, "w");
        compare(r.letters.apps, "a");
    }

    function test_defaultTakenTooTurnsThePrefixOff() {
        // apps takes "f"; files wants "f" (taken) and its default "f" is taken too.
        const r = Prefix.checkLetters({
            "prefixApps": "f"
        });
        compare(r.letters.apps, "f");
        compare(r.errors.length, 1);
        compare(r.errors[0].key, "prefixFiles");
        compare(r.errors[0].fallback, null);
        compare(r.letters.files, null);
        compare(Prefix.parse("@f x", r.letters).mode, "apps");
    }

    function test_invalidCharacterFallsBack() {
        const r = Prefix.checkLetters({
            "prefixGrep": "@",
            "prefixFiles": "F",
            "prefixWeb": "ab"
        });
        compare(r.errors.length, 3);
        compare(r.letters.grep, "g");
        compare(r.letters.files, "f");
        compare(r.letters.web, "w");
    }

    function test_digitsAreAllowed() {
        const r = Prefix.checkLetters({
            "prefixApps": "1"
        });
        compare(r.errors.length, 0);
        compare(Prefix.parse("@1 x", r.letters).mode, "apps");
    }

}
