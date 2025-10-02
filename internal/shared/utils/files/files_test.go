package files

import (
	"fmt"
	"testing"
)

func TestGetScript(t *testing.T) {
	src, err := GetLuaScript("add_player_to_room.lua")
	if err != nil {
		t.Errorf("unexpected error - %v", err)
	}
	if src == "" {
		t.Errorf("expected src to have contents - %v", src)
	}
}

func TestGetRandomWords(t *testing.T) {
	t.Run("bad length", func(t *testing.T) {
		_, err := GetThreeRandomWords(2)
		if err == nil {
			t.Error("expected error when retrievin two letter random words")
		}
	})

	for i := 3; i < 6; i++ {
		t.Run(fmt.Sprintf("retrieves %d letter words", i), func(t *testing.T) {
			length := i
			words, err := GetThreeRandomWords(length)
			if err != nil {
				t.Fatalf("expected no error when retrievin %d letter random words got %v", length, err)
			}
			if len(words) != 3 {
				t.Fatalf("expected length of 3 got %d", len(words))
			}
			if len(words[0]) != length {
				t.Errorf("expected words retrieved to have legnth of %d got %d", length, len(words[0]))
			}

			if len(words[1]) != length {
				t.Errorf("expected words retrieved to have legnth of %d got %d", length, len(words[1]))
			}

			if len(words[2]) != length {
				t.Errorf("expected words retrieved to have legnth of %d got %d", length, len(words[2]))
			}
		})
	}

	t.Run("gets new random words after two consecutive times", func(t *testing.T) {
		words, err := GetThreeRandomWords(3)
		if err != nil {
			t.Errorf("error retrieving words -%v", err)
		}
		moreWords, err := GetThreeRandomWords(3)
		if err != nil {
			t.Errorf("error retrieving words -%v", err)
		}

		for i := 0; i < 3; i++ {
			if words[i] == moreWords[i] {
				t.Errorf("expected two different words got %s and %s", words[i], moreWords[i])
			}
		}
	})
}
