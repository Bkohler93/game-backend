package game

import (
	"fmt"
	"strings"
)

type Coordinate struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

func (c *Coordinate) Equals(c2 *Coordinate) bool {
	return c.Col == c2.Col && c.Row == c2.Row
}

type LetterFoundCoordinate struct {
	Coordinate     Coordinate     `json:"coordinate"`
	GameTileStatus GameTileStatus `json:"game_tile_status"`
}
type LetterResponseStatus struct {
	UncoveredGameTiles []LetterFoundCoordinate `json:"uncovered_game_tiles"`
	KeyboardStatus     KeyboardLetterStatus    `json:"keyboard_status"`
}

type GameTileStatus int

const (
	Uncovered GameTileStatus = iota
	WordFound
)

type KeyboardLetterStatus int

const (
	InPlay KeyboardLetterStatus = iota
	AllFound
	OutOfPlay
)

type WordPlacements map[int][]*Coordinate
type WordSelections map[int]string
type FoundTiles map[Coordinate]GameTileStatus
type LetterCoordinates map[rune][]*Coordinate
type CoordinateLetters map[Coordinate]rune
type GameStateUpdate map[string]*LetterResponseStatus

func (g GameStateUpdate) ToString() string {
	s := ""
	for letter, letterResponseStatus := range g {
		s += letter + ":"
		s += fmt.Sprintf("%d,[", letterResponseStatus.KeyboardStatus)
		for _, letterFoundCoord := range letterResponseStatus.UncoveredGameTiles {
			s += fmt.Sprintf("(%d,%d)%d,", letterFoundCoord.Coordinate.Row, letterFoundCoord.Coordinate.Col, letterFoundCoord.GameTileStatus)
		}
		s = strings.TrimRight(s, ",")
		s += "]\n"
	}
	return s
}

type Gameboard struct {
	wordPlacements *WordPlacements
	wordSelections *WordSelections
	foundTiles     *FoundTiles
	letterCoords   *LetterCoordinates
	coordLetters   *CoordinateLetters
}

func (g *Gameboard) ProcessGameplayTileUncover(row int, col int) (GameStateUpdate, TurnResult) {
	newGameState := make(GameStateUpdate)
	var turnResult TurnResult

	//find if the uncovered tile houses a letter or not
	(*g.foundTiles)[Coordinate{row, col}] = Uncovered

	// go back through each placed word and update the found coordinate to WordFound if all coordinates have been found
	allFound := true
	for _, placementCoords := range *g.wordPlacements {
		wordIsFound := true
		for _, coord := range placementCoords {
			if _, ok := (*g.foundTiles)[*coord]; !ok {
				wordIsFound = false
				allFound = false
			}
		}

		if wordIsFound {
			for _, coord := range placementCoords {
				(*g.foundTiles)[*coord] = WordFound
			}
		}
	}

	// add found coordinates to new game state
	for coord, status := range *g.foundTiles {
		if letter, ok := (*g.coordLetters)[coord]; ok {
			if _, ok := newGameState[string(letter)]; !ok {
				newGameState[string(letter)] = &LetterResponseStatus{
					UncoveredGameTiles: []LetterFoundCoordinate{},
					KeyboardStatus:     OutOfPlay,
				}
			}
			newGameState[string(letter)].UncoveredGameTiles = append(newGameState[string(letter)].UncoveredGameTiles, LetterFoundCoordinate{
				Coordinate:     Coordinate{coord.Row, coord.Col},
				GameTileStatus: status,
			})
		} else {
			if _, ok := newGameState[" "]; !ok {
				newGameState[" "] = &LetterResponseStatus{
					UncoveredGameTiles: []LetterFoundCoordinate{},
					KeyboardStatus:     OutOfPlay,
				}
			}
			newGameState[" "].UncoveredGameTiles = append(newGameState[" "].UncoveredGameTiles, LetterFoundCoordinate{
				Coordinate:     Coordinate{coord.Row, coord.Col},
				GameTileStatus: status,
			})
		}
	}

	// update KeyboardStatus for each entry in new game state
	for letter, letterResponseStatus := range newGameState {
		allInstancesOfLetterFound := true

		if len((*g.letterCoords)[rune(letter[0])]) == 0 {
			letterResponseStatus.KeyboardStatus = OutOfPlay
			continue
		}
		for _, coord := range (*g.letterCoords)[rune(letter[0])] {
			if _, ok := (*g.foundTiles)[*coord]; !ok {
				allInstancesOfLetterFound = false
			}
		}

		if allInstancesOfLetterFound {
			letterResponseStatus.KeyboardStatus = AllFound
		} else {
			letterResponseStatus.KeyboardStatus = InPlay
		}
	}

	if allFound {
		turnResult = Win
	} else {
		turnResult = EndTurn
	}

	return newGameState, turnResult
}

func (g *Gameboard) ProcessGameplayWordGuess(word string) (GameStateUpdate, TurnResult) {
	newGameState := make(GameStateUpdate)
	var turnResult TurnResult

	exactWordMatch := false
	for _, wd := range *g.wordSelections {
		if word == wd {
			exactWordMatch = true
		}
	}

	// uncover any letters if the letter at index i for the guessed word is the same letter for a placed word's letter at that index
	for guessWordIdx, guessWordChar := range word {
		newGameState[string(guessWordChar)] = &LetterResponseStatus{
			UncoveredGameTiles: []LetterFoundCoordinate{},
			KeyboardStatus:     OutOfPlay,
		}
		for wordLen, checkWord := range *g.wordSelections {
			if guessWordIdx >= wordLen {
				continue
			}
			if guessWordChar == rune(checkWord[guessWordIdx]) {
				foundCoord := (*g.wordPlacements)[wordLen][guessWordIdx]
				if _, ok := (*g.foundTiles)[*foundCoord]; !ok {
					(*g.foundTiles)[*foundCoord] = Uncovered
				}
			}
		}
	}

	// go back through each placed word and update the found coordinate to WordFound if all coordinates have been found
	allFound := true
	for _, placementCoords := range *g.wordPlacements {
		wordIsFound := true
		for _, coord := range placementCoords {
			if _, ok := (*g.foundTiles)[*coord]; !ok {
				wordIsFound = false
				allFound = false
			}
		}

		if wordIsFound {
			for _, coord := range placementCoords {
				(*g.foundTiles)[*coord] = WordFound
			}
		}
	}

	// add found coordinates to new game state
	for coord, status := range *g.foundTiles {
		if letter, ok := (*g.coordLetters)[coord]; ok {
			if _, ok := newGameState[string(letter)]; !ok {
				newGameState[string(letter)] = &LetterResponseStatus{
					UncoveredGameTiles: []LetterFoundCoordinate{},
					KeyboardStatus:     OutOfPlay,
				}
			}
			newGameState[string(letter)].UncoveredGameTiles = append(newGameState[string(letter)].UncoveredGameTiles, LetterFoundCoordinate{
				Coordinate:     Coordinate{coord.Row, coord.Col},
				GameTileStatus: status,
			})
		} else {
			if _, ok := newGameState[" "]; !ok {
				newGameState[" "] = &LetterResponseStatus{
					UncoveredGameTiles: []LetterFoundCoordinate{},
					KeyboardStatus:     OutOfPlay,
				}
			}
			newGameState[" "].UncoveredGameTiles = append(newGameState[" "].UncoveredGameTiles, LetterFoundCoordinate{
				Coordinate:     Coordinate{coord.Row, coord.Col},
				GameTileStatus: status,
			})
		}
	}

	// update KeyboardStatus for each entry in new game state
	for letter, letterResponseStatus := range newGameState {
		allInstancesOfLetterFound := true

		if len((*g.letterCoords)[rune(letter[0])]) == 0 {
			letterResponseStatus.KeyboardStatus = OutOfPlay
			continue
		}
		for _, coord := range (*g.letterCoords)[rune(letter[0])] {
			if _, ok := (*g.foundTiles)[*coord]; !ok {
				allInstancesOfLetterFound = false
			}
		}

		if allInstancesOfLetterFound {
			letterResponseStatus.KeyboardStatus = AllFound
		} else {
			letterResponseStatus.KeyboardStatus = InPlay
		}
	}

	if allFound {
		turnResult = Win
	} else if exactWordMatch {
		turnResult = GoAgain
	} else {
		turnResult = EndTurn
	}

	return newGameState, turnResult
}

func (g *Gameboard) ProcessTurnDeciderWordGuess(word string) (GameStateUpdate, int) {
	newGameState := make(GameStateUpdate)
	var numTilesUncovered int

	// uncover any letters if the letter at index i for the guessed word is the same letter for a placed word's letter at that index
	for guessWordIdx, guessWordChar := range word {
		newGameState[string(guessWordChar)] = &LetterResponseStatus{
			UncoveredGameTiles: []LetterFoundCoordinate{},
			KeyboardStatus:     OutOfPlay,
		}
		for wordLen, checkWord := range *g.wordSelections {
			if guessWordIdx >= wordLen {
				continue
			}
			if guessWordChar == rune(checkWord[guessWordIdx]) {
				foundCoord := (*g.wordPlacements)[wordLen][guessWordIdx]
				if _, ok := (*g.foundTiles)[*foundCoord]; !ok {
					(*g.foundTiles)[*foundCoord] = Uncovered
					numTilesUncovered++
				}
			}
		}
	}

	// go back through each placed word and update the found coordinate to WordFound if all coordinates have been found
	for _, placementCoords := range *g.wordPlacements {
		wordIsFound := true
		for _, coord := range placementCoords {
			if _, ok := (*g.foundTiles)[*coord]; !ok {
				wordIsFound = false
			}
		}

		if wordIsFound {
			for _, coord := range placementCoords {
				(*g.foundTiles)[*coord] = WordFound
			}
		}
	}

	// add found coordinates to new game state
	for letter, letterCords := range *g.letterCoords {
		for _, coord := range letterCords {
			if status, ok := (*g.foundTiles)[*coord]; ok {
				if _, ok := newGameState[string(letter)]; !ok {
					newGameState[string(letter)] = &LetterResponseStatus{
						UncoveredGameTiles: []LetterFoundCoordinate{},
						KeyboardStatus:     OutOfPlay,
					}
				}
				newGameState[string(letter)].UncoveredGameTiles = append(newGameState[string(letter)].UncoveredGameTiles, LetterFoundCoordinate{
					Coordinate:     Coordinate{(*coord).Row, (*coord).Col},
					GameTileStatus: status,
				})
			}
		}
	}

	// update KeyboardStatus for each entry in new game state
	for letter, letterResponseStatus := range newGameState {
		allInstancesOfLetterFound := true

		if len((*g.letterCoords)[rune(letter[0])]) == 0 {
			letterResponseStatus.KeyboardStatus = OutOfPlay
			continue
		}
		for _, coord := range (*g.letterCoords)[rune(letter[0])] {
			if _, ok := (*g.foundTiles)[*coord]; !ok {
				allInstancesOfLetterFound = false
			}
		}

		if allInstancesOfLetterFound {
			letterResponseStatus.KeyboardStatus = AllFound
		} else {
			letterResponseStatus.KeyboardStatus = InPlay
		}
	}

	return newGameState, numTilesUncovered
}

func NewGameboard(wp *WordPlacements, ws *WordSelections) *Gameboard {
	foundTiles := make(FoundTiles)
	letterCoords := make(LetterCoordinates)
	coordLetters := make(CoordinateLetters)
	for wordLen, word := range *ws {
		for letterIdx, letter := range word {
			if _, ok := letterCoords[letter]; !ok {
				letterCoords[letter] = make([]*Coordinate, 0)
			}
			letterCoords[letter] = append(letterCoords[letter], (*wp)[wordLen][letterIdx])
			coordLetters[*(*wp)[wordLen][letterIdx]] = letter
		}
	}
	// letterCoords[' '] = make([]*Coordinate, 0) //will hold all coordinates for tiles uncovered that have no letters
	return &Gameboard{wp, ws, &foundTiles, &letterCoords, &coordLetters}
}
