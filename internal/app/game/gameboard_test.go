package game

import (
	"testing"
)

func TestProcessTurnDeciderGuess(t *testing.T) {
	gb := NewGameboard(&WordPlacements{
		3: []*Coordinate{
			{0, 0},
			{0, 1},
			{0, 2},
		},
		4: []*Coordinate{
			{2, 0},
			{2, 1},
			{2, 2},
			{2, 3},
		},
		5: []*Coordinate{
			{4, 0},
			{4, 1},
			{4, 2},
			{4, 3},
			{4, 4},
		},
	}, &WordSelections{
		3: "tit",
		4: "ball",
		5: "plant",
	})

	guessWord := "taint"
	numUniqueLetters := 4
	gameStateUpdate, numUncovered := gb.ProcessTurnDeciderWordGuess(guessWord)
	expectedUncovered := 4

	if numUncovered != expectedUncovered {
		t.Fatalf("expected %d to be uncovered got %d", expectedUncovered, numUncovered)
	}

	if len(gameStateUpdate) != numUniqueLetters {
		t.Fatalf("expected state update to include updates for %d letters, got %d", numUniqueLetters, len(gameStateUpdate))
	}

	for k, v := range gameStateUpdate {
		switch k {
		case "t":
			if (*v).KeyboardStatus != InPlay {
				t.Errorf("expected 't' to still be InPlay after guess got %v", (*v).KeyboardStatus)
			}
		case "a":
			if (*v).KeyboardStatus != InPlay {
				t.Errorf("expected 'a' to still be InPlay after guess got %v", (*v).KeyboardStatus)
			}
		case "i":
			if (*v).KeyboardStatus != InPlay {
				t.Errorf("expected 'i' to still be InPlay after guess got %v", (*v).KeyboardStatus)
			}
		case "n":
			if (*v).KeyboardStatus != AllFound {
				t.Errorf("expected 'n' to AllFound after guess got %v", (*v).KeyboardStatus)
			}
		default:
			t.Errorf("got unknown letter '%s'. should not be in results", k)
		}
	}

	for letter, letterResponseStatus := range gameStateUpdate {
		numUncoveredTiles := len(letterResponseStatus.UncoveredGameTiles)
		switch letter {
		case "t":
			if numUncoveredTiles != 2 {
				t.Errorf("expected to find 2 uncovered tiles from the 't' guess got - %d", numUncoveredTiles)
			}
		case "a":
			if numUncoveredTiles != 1 {
				t.Errorf("expected to find 1 uncovered tiles from the 't' guess got - %d", numUncoveredTiles)
			}
		case "i":
			if numUncoveredTiles != 0 {
				t.Errorf("expected to find 0 uncovered tiles from the 't' guess got - %d", numUncoveredTiles)
			}
		case "n":
			if numUncoveredTiles != 1 {
				t.Errorf("expected to find 1 uncovered tiles from the 't' guess got - %d", numUncoveredTiles)
			}
		}
	}

	guessWord = "pit"
	gameStateUpdate, numUncovered = gb.ProcessTurnDeciderWordGuess(guessWord)
	expectedUncovered = 3

	if expectedUncovered != numUncovered {
		t.Errorf("expected %d to be uncovered got %d", expectedUncovered, numUncovered)
	}
	if gameStateUpdate["t"].KeyboardStatus != AllFound {
		t.Errorf("expected all t's to be found got %d", gameStateUpdate["t"].KeyboardStatus)
	}

	guessWord = "shit"
	_, numUncovered = gb.ProcessTurnDeciderWordGuess(guessWord)
	expectedUncovered = 0

	if numUncovered != expectedUncovered {
		t.Errorf("expected %d uncovered got %d", expectedUncovered, numUncovered)
	}
}

func TestProcesssGameplayGuesses(t *testing.T) {
	gb := NewGameboard(&WordPlacements{
		3: []*Coordinate{
			{0, 0},
			{0, 1},
			{0, 2},
		},
		4: []*Coordinate{
			{2, 0},
			{2, 1},
			{2, 2},
			{2, 3},
		},
		5: []*Coordinate{
			{4, 0},
			{4, 1},
			{4, 2},
			{4, 3},
			{4, 4},
		},
	}, &WordSelections{
		3: "tit",
		4: "ball",
		5: "plant",
	})

	guessWord := "taint"
	numUniqueLetters := 4
	gameStateUpdate, turnResult := gb.ProcessGameplayWordGuess(guessWord)

	if turnResult != EndTurn {
		t.Errorf("expected 'EndTurn' result got %#v", turnResult)
	}

	if len(gameStateUpdate) != numUniqueLetters {
		t.Errorf("expected %d results in the update got %d", numUniqueLetters, len(gameStateUpdate))
	}

	for k, v := range gameStateUpdate {
		switch k {
		case "t":
			if (*v).KeyboardStatus != InPlay {
				t.Errorf("expected 't' to still be InPlay after guess got %v", (*v).KeyboardStatus)
			}
			c1 := Coordinate{0, 0}
			c2 := Coordinate{4, 4}

			for _, v := range (*v).UncoveredGameTiles {
				if (v.Coordinate.Row != c1.Row && v.Coordinate.Col != c1.Col) && (v.Coordinate.Row != c2.Row && v.Coordinate.Col != c2.Col) {
					t.Errorf("unexpected coordinate for letter 't' :  %#v", v.Coordinate)
				}
			}
		case "a":
			if (*v).KeyboardStatus != InPlay {
				t.Errorf("expected 'a' to still be InPlay after guess got %v", (*v).KeyboardStatus)
			}
		case "i":
			if (*v).KeyboardStatus != InPlay {
				t.Errorf("expected 'i' to still be InPlay after guess got %v", (*v).KeyboardStatus)
			}
		case "n":
			if (*v).KeyboardStatus != AllFound {
				t.Errorf("expected 'n' to AllFound after guess got %v", (*v).KeyboardStatus)
			}
		default:
			t.Errorf("got unknown letter '%s'. should not be in results", k)
		}
	}

	guessWord = "plant"
	_, turnResult = gb.ProcessGameplayWordGuess(guessWord)
	if turnResult != GoAgain {
		t.Errorf("expected go again after guessing exact matching word got %#v", turnResult)
	}

	gameStateUpdate, turnResult = gb.ProcessGameplayTileUncover(0, 1)

	if turnResult != EndTurn {
		t.Errorf("expected tile uncovering to result in 'EndTurn' got %#v", turnResult)
	}

	if gameStateUpdate["i"].KeyboardStatus != AllFound {
		t.Errorf("expected KeyboardStatus for 'i' to be AllFound got %#v", gameStateUpdate["i"].KeyboardStatus)
	}

	gameStateUpdate, _ = gb.ProcessGameplayTileUncover(3, 3)
	if gameStateUpdate[" "].UncoveredGameTiles[0].Coordinate.Col != 3 || gameStateUpdate[" "].UncoveredGameTiles[0].Coordinate.Row != 3 {
		t.Errorf("expected empty tile to have an entry in the game state's UncoveredGameTiles field with coordiante (3,3) got %#v", gameStateUpdate[" "].UncoveredGameTiles[0].Coordinate)
	}

	gameStateUpdate, _ = gb.ProcessGameplayTileUncover(3, 4)
	if len(gameStateUpdate[" "].UncoveredGameTiles) != 2 {
		t.Errorf("expected 2 uhcovered game tiles to have an empty string, got %d", len(gameStateUpdate[" "].UncoveredGameTiles))
	}

	guessWord = "tit"
	_, _ = gb.ProcessGameplayWordGuess(guessWord)
	guessWord = "ball"
	_, turnResult = gb.ProcessGameplayWordGuess(guessWord)
	if turnResult != Win {
		t.Errorf("guessing all exact words should end in a Win - got %v", turnResult)
	}
}

func TestProcesssGameplayGuessesEndWithUncoverTile(t *testing.T) {
	gb := NewGameboard(&WordPlacements{
		3: []*Coordinate{
			{0, 0},
			{0, 1},
			{0, 2},
		},
		4: []*Coordinate{
			{2, 0},
			{2, 1},
			{2, 2},
			{2, 3},
		},
		5: []*Coordinate{
			{4, 0},
			{4, 1},
			{4, 2},
			{4, 3},
			{4, 4},
		},
	}, &WordSelections{
		3: "tit",
		4: "ball",
		5: "plant",
	})

	guessWord := "taint"
	numUniqueLetters := 4
	gameStateUpdate, turnResult := gb.ProcessGameplayWordGuess(guessWord)

	if turnResult != EndTurn {
		t.Errorf("expected 'EndTurn' result got %#v", turnResult)
	}

	if len(gameStateUpdate) != numUniqueLetters {
		t.Errorf("expected %d results in the update got %d", numUniqueLetters, len(gameStateUpdate))
	}

	for k, v := range gameStateUpdate {
		switch k {
		case "t":
			if (*v).KeyboardStatus != InPlay {
				t.Errorf("expected 't' to still be InPlay after guess got %v", (*v).KeyboardStatus)
			}
			c1 := Coordinate{0, 0}
			c2 := Coordinate{4, 4}

			for _, v := range (*v).UncoveredGameTiles {
				if (v.Coordinate.Row != c1.Row && v.Coordinate.Col != c1.Col) && (v.Coordinate.Row != c2.Row && v.Coordinate.Col != c2.Col) {
					t.Errorf("unexpected coordinate for letter 't' :  %#v", v.Coordinate)
				}
			}
		case "a":
			if (*v).KeyboardStatus != InPlay {
				t.Errorf("expected 'a' to still be InPlay after guess got %v", (*v).KeyboardStatus)
			}
		case "i":
			if (*v).KeyboardStatus != InPlay {
				t.Errorf("expected 'i' to still be InPlay after guess got %v", (*v).KeyboardStatus)
			}
		case "n":
			if (*v).KeyboardStatus != AllFound {
				t.Errorf("expected 'n' to AllFound after guess got %v", (*v).KeyboardStatus)
			}
		default:
			t.Errorf("got unknown letter '%s'. should not be in results", k)
		}
	}

	guessWord = "plant"
	_, turnResult = gb.ProcessGameplayWordGuess(guessWord)
	if turnResult != GoAgain {
		t.Errorf("expected go again after guessing exact matching word got %#v", turnResult)
	}

	gameStateUpdate, turnResult = gb.ProcessGameplayTileUncover(0, 1)

	if turnResult != EndTurn {
		t.Errorf("expected tile uncovering to result in 'EndTurn' got %#v", turnResult)
	}

	if gameStateUpdate["i"].KeyboardStatus != AllFound {
		t.Errorf("expected KeyboardStatus for 'i' to be AllFound got %#v", gameStateUpdate["i"].KeyboardStatus)
	}

	gameStateUpdate, _ = gb.ProcessGameplayTileUncover(3, 3)
	if gameStateUpdate[" "].UncoveredGameTiles[0].Coordinate.Col != 3 || gameStateUpdate[" "].UncoveredGameTiles[0].Coordinate.Row != 3 {
		t.Errorf("expected empty tile to have an entry in the game state's UncoveredGameTiles field with coordiante (3,3) got %#v", gameStateUpdate[" "].UncoveredGameTiles[0].Coordinate)
	}

	gameStateUpdate, _ = gb.ProcessGameplayTileUncover(3, 4)
	if len(gameStateUpdate[" "].UncoveredGameTiles) != 2 {
		t.Errorf("expected 2 uhcovered game tiles to have an empty string, got %d", len(gameStateUpdate[" "].UncoveredGameTiles))
	}

	guessWord = "tit"
	_, _ = gb.ProcessGameplayWordGuess(guessWord)

	_, _ = gb.ProcessGameplayTileUncover(2, 0)
	_, _ = gb.ProcessGameplayTileUncover(2, 2)
	_, turnResult = gb.ProcessGameplayTileUncover(2, 3)
	if turnResult != Win {
		t.Errorf("expected win after uncovering final game tiles got %v", turnResult)
	}
}
