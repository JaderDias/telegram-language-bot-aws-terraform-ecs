package main

import (
	"fmt"
	"testing"
)

type testCase struct {
	input    string
	expected Word
}

// Test extractDefinition from a string using regular expressions
func TestExtractDefinition(t *testing.T) {
	// initialise test cases
	testCases := []testCase{
		{
			input: "opzicht===Pronunciation===\n* {{audio|nl|Nl-opzicht.ogg|Audio}}\n* {{hyphenation|nl|op|zicht}}\n\n===Noun===\n{{nl-noun|n|-en|opzichtje}}\n\n# {{l|en|supervision}}\n# {{l|en|relation}}, {{l|en|regard}}\n#: ''In welk '''opzicht'''?'' - In what regard?\n\n====Derived terms====\n* {{l|nl|opzichter}}\n* {{l|nl|ten opzichte van}}",
			expected: Word{
				title:            "opzicht",
				grammaticalClass: "Noun",
				mainDefinition:   "supervision",
			},
		}, {
			input: "priesterschap===Pronunciation===\n* {{IPA|nl|/ˈpris.tərˌsxɑp/}}\n* {{audio|nl|Nl-priesterschap.ogg|Audio}}\n* {{hyphenation|nl|pries|ter|schap}}\n\n===Noun===\n{{nl-noun|f|-en|-}}\n\n# {{l|en|priesthood}} (referring to priests as a whole)\n\n===Noun===\n{{nl-noun|n|-en|-}}\n\n# [[priesthood]] {{gloss|state of being a priest}}\n# {{lb|nl|Roman Catholicism}} one of the three major orders of the {{l|en|Roman Catholic Church}} - the {{l|en|priestly}} order",
			expected: Word{
				title:            "priesterschap",
				grammaticalClass: "Noun",
				mainDefinition:   "priesthood (referring to priests as a whole)",
			},
		}, {
			input: "rendre la monnaie de sa pièce===Pronunciation===\n* {{audio|fr|LL-Q150 (fra)-WikiLucas00-rendre la monnaie de sa pièce.wav|Audio}}\n\n===Verb===\n{{fr-verb}}\n\n# {{indtr|fr|à}} to give someone a [[taste of one's own medicine|taste of their own medicine]], to [[retaliate]], to [[pay back in someone's own coin]], to [[pay back]] [[in kind]]\n#: {{syn|fr|rendre la pareille}}",
			expected: Word{
				title:            "rendre la monnaie de sa pièce",
				grammaticalClass: "Verb",
				mainDefinition:   "to give someone a taste of their own medicine, to retaliate, to pay back in someone's own coin, to pay back in kind",
			},
		},
	}

	for _, testCase := range testCases {
		expected := fmt.Sprintf("%#v", testCase.expected)
		output := Parse(testCase.input)
		actual := fmt.Sprintf("%#v", output)
		if expected != actual {
			t.Errorf("expected %s, actual %s", expected, actual)
		}
	}
}
