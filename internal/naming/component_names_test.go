package naming

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsValidComponentNameMatchesPattern is the drift guard.
//
// [ComponentNamePattern] and [IsValidComponentName] are two statements of one
// rule — the string exists to be quoted in error messages, the function to be
// run — so a change to either that the other does not follow is a silent bug.
// Running the compiled pattern against the predicate over every rune the rule
// could plausibly meet, plus the strings whose regexp behavior is easy to get
// wrong, keeps them honest.
func TestIsValidComponentNameMatchesPattern(t *testing.T) {
	pattern := regexp.MustCompile(ComponentNamePattern)

	t.Run("every single rune agrees", func(t *testing.T) {
		// Through the Latin, Greek, Cyrillic, Hebrew and Arabic blocks, which is
		// where a letter that unicode.IsLetter accepts and the ASCII-only rule
		// rejects actually turns up, plus a CJK sample.
		for r := rune(0); r <= 0x0700; r++ {
			s := string(r)
			require.Equal(t, pattern.MatchString(s), IsValidComponentName(s),
				"disagreement on rune %U (%q)", r, s)
		}
		for _, r := range []rune{'宠', '物', '中', '日', 'é', '́', 'Ａ'} {
			s := string(r)
			require.Equal(t, pattern.MatchString(s), IsValidComponentName(s),
				"disagreement on rune %U (%q)", r, s)
		}
	})

	t.Run("whole strings agree", func(t *testing.T) {
		names := []string{
			"", "Pet", "pet", "PET", "Pet1", "1Pet", "0", "_", "-", ".",
			"pkg.v1-Pet_2", "...", "___", "---",
			"pkg/Pet", "pet~summary", "Pet@v1", "Pet Name", "Response[User]",
			"Pét", "café.Order", "宠物", "Ünïcödé",
			// A trailing newline is the classic way a "$"-anchored pattern and a
			// hand-rolled loop part company.
			"Pet\n", "\nPet", "Pet\n\n", "\n",
			"Pet\t", "Pet\r", "Pet\x00",
		}
		for _, name := range names {
			assert.Equal(t, pattern.MatchString(name), IsValidComponentName(name),
				"disagreement on %q", name)
		}
	})

	t.Run("predicate agrees with the pattern per character", func(t *testing.T) {
		for r := rune(0); r <= 0x0700; r++ {
			assert.Equal(t, pattern.MatchString(string(r)), IsComponentNameChar(r),
				"disagreement on rune %U", r)
		}
	})
}

// TestIsComponentNameChar covers the charset boundaries directly, so a failure
// names the character rather than pointing at a sweep.
func TestIsComponentNameChar(t *testing.T) {
	allowed := []rune{'a', 'm', 'z', 'A', 'M', 'Z', '0', '5', '9', '.', '-', '_'}
	for _, r := range allowed {
		assert.True(t, IsComponentNameChar(r), "%q should be allowed", r)
	}

	rejected := []rune{
		'/', '~', '@', '!', '#', '$', '%', '&', '*', '+', '=', '?', ':', ';',
		'\'', '"', '(', ')', '[', ']', '{', '}', '<', '>', ',', '|', '\\', '^',
		'`', ' ', '\t', '\n',
		// Letters and digits that are not ASCII.
		'é', '宠', '中', 'Ω', 'Я', '٣',
	}
	for _, r := range rejected {
		assert.False(t, IsComponentNameChar(r), "%q should be rejected", r)
	}
}

// TestIsValidComponentName covers the string-level rule, including the empty
// name that the pattern's "+" quantifier excludes.
func TestIsValidComponentName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple name", "Pet", true},
		{"all the allowed punctuation", "pkg.v1-Pet_2", true},
		{"digits only", "12345", true},
		{"punctuation only", "._-", true},
		{"empty is rejected by the + quantifier", "", false},
		{"whitespace", "   ", false},
		{"slash", "pkg/Pet", false},
		{"tilde", "pet~summary", false},
		{"accented letter", "Pét", false},
		{"CJK", "宠物", false},
		{"one bad character in a long name", "PetPetPetPet@", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidComponentName(tt.input))
		})
	}
}

func ExampleIsValidComponentName() {
	fmt.Println(IsValidComponentName("pkg.v1-Pet"))
	fmt.Println(IsValidComponentName("pkg/Pet"))
	// Output:
	// true
	// false
}
