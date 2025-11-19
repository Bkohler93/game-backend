package files

import (
	"bufio"
	"embed"
	"fmt"
	"math/rand"
)

//go:embed wordlist/*.txt
var WordListsFS embed.FS

var wordLists map[int][]string

func GetThreeRandomWords(length int) ([]string, error) {
	if wordLists == nil {
		wordLists = make(map[int][]string)
	}
	if wordLists[length] == nil {
		filename := fmt.Sprintf("wordlist/%d_letters.txt", length)
		f, err := WordListsFS.Open(filename)
		if err != nil {
			return nil, err
		}
		wordLists[length] = make([]string, 0, 100)
		s := bufio.NewScanner(f)
		for s.Scan() {
			txt := s.Text()
			if txt == "" {
				break
			}
			wordLists[length] = append(wordLists[length], txt)
		}
	}

	words := wordLists[length]
	if len(words) < 3 {
		return nil, fmt.Errorf("not enough words to choose from")
	}

	indices := rand.Perm(len(words))[:3]
	return []string{words[indices[0]], words[indices[1]], words[indices[2]]}, nil
}
