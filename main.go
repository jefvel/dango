package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/thoj/go-ircevent"
)

type azInstance struct {
	answer      int
	currentLow  int
	currentHigh int
    correctGuesses map[string]int
    incorrectGuesses map[string]int
}

func newAZ(words []string) (az *azInstance) {
	az = &azInstance{}
	az.answer = rand.Intn(len(words))
	if az.answer == 0 {
		az.answer = 1
	} else if az.answer == len(words)-1 {
		az.answer = len(words) - 2
	}
	az.currentLow = 0
	az.currentHigh = len(words) - 1
    az.correctGuesses = make(map[string]int)
    az.incorrectGuesses = make(map[string]int)
	return
}

func extractAZGuess(message string) bool {
	for _, char := range message {
		if !unicode.IsLetter(char) {
			return false
		}
	}

	return true
}

func main() {
	server := flag.String("s", "", "irc server to connect to")
	channel := flag.String("c", "", "irc channel to connect to")
	wordlist := flag.String("w", "", "wordlist file to use")

	flag.Parse()

	log.Printf("using server %s", *server)
	log.Printf("using channel %s", *channel)
	log.Printf("using wordlist %s", *wordlist)

	rand.Seed(time.Now().UTC().UnixNano())
	var az *azInstance

	file, err := os.Open(*wordlist)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	words := make([]string, 0, 1000)
	scanner := bufio.NewScanner(file)

	log.Print("scanning for words")
	for scanner.Scan() {
		word := scanner.Text()
		word = strings.Replace(word, "!", "", -1)
		word = strings.Replace(word, "$", "", -1)
		word = strings.Replace(word, "+", "", -1)
		word = strings.Replace(word, "^", "", -1)
		word = strings.Replace(word, "&", "", -1)
		words = append(words, word)
	}
	log.Print("done scanning")

	ircnick1 := "dango"
	irccon := irc.IRC(ircnick1, "Dan Go")
	irccon.VerboseCallbackHandler = true
	irccon.Debug = true
	irccon.UseTLS = true
	irccon.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	irccon.AddCallback("001", func(e *irc.Event) { irccon.Join(*channel) })
	irccon.AddCallback("366", func(e *irc.Event) {})
	irccon.AddCallback("PRIVMSG", func(e *irc.Event) {
		if e.Message() == "WOW" {
			irccon.Privmsgf(*channel, "WOW")
		} else if e.Message() == "!az" {
			if az != nil {
				irccon.Privmsgf(*channel, "game already in progress. keep guessing... %s - %s", words[az.currentLow], words[az.currentHigh])
			} else {
				az = newAZ(words)
				irccon.Privmsgf(*channel, "starting a new AZ game. Start guessing the word... %s - %s", words[az.currentLow], words[az.currentHigh])
			}
		} else if e.Message() == "!az stop" {
			if az != nil {
                loseString := resultString(az, words[az.answer], e.Nick, false)
				irccon.Privmsgf(*channel, loseString)
				az = nil
			} else {
				az = newAZ(words)
				irccon.Privmsgf(*channel, "There's no AZ game in progress ya dummy")
			}
		} else if az != nil {
			guess := e.Message()
			guessValid := extractAZGuess(guess)

			if guessValid {
                guesser := e.Nick
				currentLow := words[az.currentLow]
				currentHigh := words[az.currentHigh]
				if guess > currentLow && guess < currentHigh {
					//this is a valid guess range wise. just gotta see that it exists in the word list
					log.Printf("guess %s is within range", guess)
					foundIndex := sort.Search(len(words), func(i int) bool {
						log.Printf("index in search %d, guess: %s, current: %s result of (%s <= %s): %t", i, guess, words[i], guess, words[i], guess <= words[i])
						return guess <= words[i]
					})

					log.Printf("found index %d and len is %d", foundIndex, len(words))

					if foundIndex != len(words) && words[foundIndex] == guess {
						//this is a valid guess that exists
                        az.correctGuesses[guesser]++
						if foundIndex < az.answer {
							az.currentLow = foundIndex
							irccon.Privmsgf(*channel, "%s is not right, but closer! %s - %s", words[foundIndex], words[az.currentLow], words[az.currentHigh])
						} else if foundIndex > az.answer {
							az.currentHigh = foundIndex
							irccon.Privmsgf(*channel, "%s is not right, but closer! %s - %s", words[foundIndex], words[az.currentLow], words[az.currentHigh])
						} else {
                            winString := resultString(az, words[foundIndex], guesser, true)
                            irccon.Privmsgf(*channel, winString)
							az = nil
						}
					} else {
						irccon.Privmsgf(*channel, "%s is not a valid word. try again", guess)
                        az.incorrectGuesses[guesser]++
					}
				}
			}
		}
	})
	err = irccon.Connect(*server)
	if err != nil {
		fmt.Printf("Err %s", err)
		return
	}

	irccon.Loop()
}

func resultString(az *azInstance, answer string, guesser string, success bool) string {
    var totalCorrectGuesses int
    var correctGuessesStr string
    for nick, value := range az.correctGuesses {
        correctGuessesStr += fmt.Sprintf("%s (%d), ", nick, value)
        totalCorrectGuesses += value
    }
    if totalCorrectGuesses == 1 {
        correctGuessesStr = fmt.Sprintf("1 correct guess: %s", correctGuessesStr)
    } else {
        correctGuessesStr = fmt.Sprintf("%d correct guesses: %s", totalCorrectGuesses, correctGuessesStr)
    }
    var totalIncorrectGuesses int
    var incorrectGuessesStr string
    for _, value := range az.incorrectGuesses {
        totalIncorrectGuesses += value
    }
    if totalIncorrectGuesses == 1 {
        incorrectGuessesStr = fmt.Sprintf("1 incorrect guess")
    } else {
        incorrectGuessesStr = fmt.Sprintf("%d incorrect guesses", totalIncorrectGuesses)
    }
    if success {
        return fmt.Sprintf("WOW %s you won, %s was the right word! %s %s", guesser, answer, correctGuessesStr, incorrectGuessesStr)
    } else {
        return fmt.Sprintf("So sad that you couldn't solve it yourself. The word was %s. %s %s", answer, correctGuessesStr, incorrectGuessesStr)
    }
}
