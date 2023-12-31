package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
)

var (
	cliName    string  = "letterboxed"
	lettersPtr *string = flag.String("letters", "", "comma-separated list of letters on each side of the box")
	trieMode   bool    = false

	// Hardcoded repl commands
	commands = map[string]interface{}{
		"help":  displayHelp,
		"clear": clearScreen,
	}
)

type gameState struct {
	groupings []string
	words     []string
}

type trie struct {
	// If true, then this trie is a complete word, even if this trie has children.
	complete bool
	children map[rune]*trie
}

func loadWords() (*trie, error) {
	f, err := os.Open("valid_words.txt")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	baseTrie := trie{children: make(map[rune]*trie)}
	for scanner.Scan() {
		word := scanner.Text()
		if strings.ContainsAny(word, "1234567890-'.!@#$%^&*()") {
			continue
		}
		t := &baseTrie
		for i, r := range strings.ToUpper(word) {
			nextTrie, found := t.children[r]
			if !found {
				complete := i == len(word)-1
				//newTrie := new(trie)
				t.children[r] = &trie{
					complete: complete,
					children: make(map[rune]*trie),
				}
				nextTrie = t.children[r]
			}
			t = nextTrie
		}
	}
	return &baseTrie, nil
}

// printPrompt displays the repl prompt at the start of each loop
func printPrompt() {
	pre := cliName
	if trieMode {
		pre = "trie"
	}
	fmt.Print(pre, "> ")
}

// displayHelp informs the user about our hardcoded functions
func displayHelp() {
	fmt.Printf(
		"Welcome to %v! These are the available commands: \n",
		cliName,
	)
	fmt.Println("help    - Show available commands")
	fmt.Println("clear   - Clear the terminal screen")
	fmt.Println("exit    - Closes your connection")
}

// clearScreen clears the terminal screen
func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		fmt.Printf("%v", err)
	}
}

func (t *trie) sortedChildren() []string {
	sc := make([]string, 0, len(t.children))
	for k := range t.children {
		sc = append(sc, string(k))
	}
	sort.Strings(sc)
	return sc
}

// exploreTrie lets us explore the dictionary trie.
func (t *trie) explore(text string, currWord string) (string, error) {
	//handleInvalidCmd(text)
	if text == ".reset" {
		return "", nil
	}
	if text == ".valid" {
		words := t.allValidWords(currWord)
		end := min(10, len(words))
		fmt.Printf("There are %d valid words. Here are the first 10: %v\n", len(words), words[:end])
		return currWord, nil
	}
	newWord := strings.ToUpper(currWord + text)
	currTrie := t
	for _, ch := range newWord {
		next, found := currTrie.children[ch]
		if !found {
			return currWord, fmt.Errorf("%s (%c) is not in the dictionary, try again", newWord, ch)
		}
		currTrie = next
	}
	children := []string{}
	for child := range currTrie.children {
		children = append(children, string(child))
	}
	if len(children) == 0 {
		fmt.Printf("Word: %s has no more children. Starting over.\n", newWord)
		return "", nil
	}
	sort.Strings(children)
	words := t.allValidWords(newWord)
	end := min(10, len(words))
	fmt.Printf(" - Chars so far: %s\n - Children: %v\n - Valid words: %v\n", newWord, children, words[:end])
	return newWord, nil
}

func (t *trie) isWordValid(word string) bool {
	curr := t
	for _, ch := range word {
		next, found := curr.children[ch]
		if !found {
			return false
		}
		curr = next
	}
	return true
}

func (t *trie) allValidWords(pre string) []string {
	curr := t
	for _, ch := range pre {
		next, found := curr.children[ch]
		if !found {
			return []string{}
		}
		curr = next
	}
	var helper func(*trie) []string
	helper = func(curr *trie) []string {
		var acc []string
		for _, child := range curr.sortedChildren() {
			ch := []rune(child)[0]
			next := curr.children[ch]
			if len(next.children) == 0 {
				acc = append(acc, string(ch))
				continue
			}
			// A word may be complete, but still have further children.
			if next.complete {
				acc = append(acc, string(ch))
			}
			words := helper(next)
			for _, w := range words {
				acc = append(acc, string(ch)+w)
			}
		}
		return acc
	}
	var words []string
	for _, w := range helper(curr) {
		words = append(words, pre+w)
	}

	return words
}

// cleanInput preprocesses input to the db repl
func cleanInput(text string) string {
	output := strings.TrimSpace(text)
	output = strings.ToLower(output)
	return output
}

func validateFlags() {
	if len(*lettersPtr) == 0 {
		fmt.Println("Error: No letters provided. Use --letters to configure the letterboxed game.")
		os.Exit(1)
	}
	groupings := strings.Split(strings.ToUpper(*lettersPtr), ",")
	if len(groupings) != 4 {
		fmt.Printf("Error: %d grouping(s) provided: %v. 4 groupings of 3 letters each are required.\n", len(groupings), groupings)
		os.Exit(1)
	}
	allLetters := make(map[rune]int)
	for _, group := range groupings {
		if len(group) != 3 {
			fmt.Printf("Error: %d grouping(s) provided: %v. 4 groupings of 3 letters each are required.\n", len(groupings), groupings)
			os.Exit(1)
		}
		for _, ch := range group {
			_, found := allLetters[ch]
			if found {
				fmt.Printf("Error: Duplicate char '%c' found. Letters must consist of 12 unique characters: %v\n", ch, groupings)
				os.Exit(1)
			}
			allLetters[ch] = 1
		}
	}
}

func validWordHelper(currTrie *trie, letters string, groupings []string, lastActiveGroup int) []string {
	acc := []string{}
	for i, group := range groupings {
		if i == lastActiveGroup {
			continue
		}
		for _, ch := range group {
			next, found := currTrie.children[ch]
			if !found {
				continue
			}
			if next.complete && len(letters) >= 2 {
				acc = append(acc, letters+string(ch))
			}
			acc = append(acc, validWordHelper(next, letters+string(ch), groupings, i)...)
		}
	}
	return acc
}

func validWords(dictionary *trie, game gameState) []string {
	return validWordHelper(dictionary, "", game.groupings, -1)
}

func validWordsStartingWith(dictionary *trie, game gameState, prefix string) []string {
	ch := []rune(prefix)[0]
	activeGroup := -1
	for i, group := range game.groupings {
		if strings.Contains(group, prefix) {
			activeGroup = i
			break
		}
	}
	if activeGroup == -1 {
		// `char` is not a valid character within the confines of the game.
		log.Fatalf("char '%s' was provided, but is not in the valid groupings: %v", prefix, game.groupings)
		return []string{}
	}
	return validWordHelper(dictionary.children[ch], prefix, game.groupings, activeGroup)
}

func isGameComplete(words []string) bool {
	letters := make(map[string]int)
	for _, w := range words {
		for _, ch := range w {
			letters[string(ch)] = 1
			if len(letters) == 12 {
				return true
			}
		}
	}
	return len(letters) == 12
}

func numberOfSolutions(dictionary *trie, game gameState) int {
	var helper func([]string) int
	helper = func(words []string) int {
		if isGameComplete(words) {
			fmt.Printf("%v: %v\n", words, isGameComplete(words))
			//reader := bufio.NewScanner(os.Stdin)
			//reader.Scan()
		}
		lastWord := words[len(words)-1]
		lastLetter := string(lastWord[len(lastWord)-1])
		nextWords := validWordsStartingWith(dictionary, game, lastLetter)
		if len(words) == 3 {
			if isGameComplete(words) {
				return 1
			}
			return 0
		}
		acc := 0
		for _, next := range nextWords {
			if slices.Contains(words, next) {
				continue
			}
			acc += helper(append(words, next))
		}
		return acc
	}
	acc := 0
	words := validWords(dictionary, game)
	byLengthThenAlphabetically := func(i int, j int) bool {
		x := words[i]
		y := words[j]
		deltaLength := len(x) - len(y)

		return deltaLength > 0 || (deltaLength == 0 && x < y)
	}
	sort.Slice(words, byLengthThenAlphabetically)

	for _, w := range words {
		acc += helper([]string{w})
	}
	return acc
}

func repl() {
	dictionary, err := loadWords()
	if err != nil {
		fmt.Printf("Error: %v", err)
		return
	}

	groupings := strings.Split(strings.ToUpper(*lettersPtr), ",")
	game := gameState{groupings: groupings}

	fmt.Printf("Letterboxed helper initiated!\n - Letters: %v\n - Solutions: %d\n",
		strings.ToUpper(*lettersPtr),
		numberOfSolutions(dictionary, game))
	currWord := ""
	reader := bufio.NewScanner(os.Stdin)
	printPrompt()
	for reader.Scan() {
		text := cleanInput(reader.Text())
		if command, exists := commands[text]; exists {
			// Call a hardcoded function
			command.(func())()
		} else if strings.EqualFold("exit", text) {
			// Close the program on the exit command
			return
		} else if strings.EqualFold("trie", text) {
			trieMode = !trieMode
			if trieMode {
				fmt.Println("Trie mode activated")
			}
		} else {
			if trieMode {
				currWord, err = dictionary.explore(text, currWord)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
				}
			} else {
				// Pass the command to the parser
				// handleCmd(text)
				if len(text) == 1 {
					validWords := validWordsStartingWith(dictionary, game, strings.ToUpper(text))
					byLengthThenAlphabetically := func(i int, j int) bool {
						x := validWords[i]
						y := validWords[j]
						deltaLength := len(x) - len(y)

						return deltaLength > 0 || (deltaLength == 0 && x < y)
					}

					sort.Slice(validWords, byLengthThenAlphabetically)
					end := min(10, len(validWords))
					fmt.Printf("There are %d valid words starting with '%s': %v\n", len(validWords), text, validWords[:end])
				}
			}
		}
		printPrompt()
	}
}

func main() {
	flag.Parse()
	validateFlags()
	repl()
	// Print an additional line if we encountered an EOF character
	fmt.Println()
}
