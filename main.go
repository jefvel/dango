package main

import (
	"bufio"
	"crypto/tls"
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

const channel = "#pallkars"
const serverssl = "irc.boxbox.org:6697"

type azInstance struct {
	answer      int
	currentLow  int
	currentHigh int
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
	rand.Seed(time.Now().UTC().UnixNano())
	var az *azInstance

	file, err := os.Open("/home/tobbe/wordlist/International/3of6game.txt")
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
	irccon.AddCallback("001", func(e *irc.Event) { irccon.Join(channel) })
	irccon.AddCallback("366", func(e *irc.Event) {})
	irccon.AddCallback("PRIVMSG", func(e *irc.Event) {
		if e.Message() == "!az" {
			if az != nil {
				irccon.Privmsgf(channel, "game already in progress. keep guessing... %s - %s", words[az.currentLow], words[az.currentHigh])
			} else {
				az = newAZ(words)
				irccon.Privmsgf(channel, "starting a new AZ game. Start guessing the word... %s - %s", words[az.currentLow], words[az.currentHigh])
			}
		} else if az != nil {
			guess := e.Message()
			guessValid := extractAZGuess(guess)

			if guessValid {
				currentLow := words[az.currentLow]
				currentHigh := words[az.currentHigh]
				if guess > currentLow && guess < currentHigh {
					//this is a valid guess range wise. just gotta see that it exists in the word list
					log.Printf("guess %s is within range", guess)
					foundIndex := sort.Search(len(words), func(i int) bool {
						log.Printf("index in search %d, guess: %s, current: %s result of (%s <= %s): %t", i, guess, words[i], guess, words[i], guess >= words[i])
						return guess <= words[i]
					})

					log.Printf("found index %d and len is %d", foundIndex, len(words))

					if foundIndex != len(words) && words[foundIndex] == guess {
						//this is a valid guess that exists
						if foundIndex < az.answer {
							az.currentLow = foundIndex
							irccon.Privmsgf(channel, "%s is not right, but closer! %s - %s", words[foundIndex], words[az.currentLow], words[az.currentHigh])
						} else if foundIndex > az.answer {
							az.currentHigh = foundIndex
							irccon.Privmsgf(channel, "%s is not right, but closer! %s - %s", words[foundIndex], words[az.currentLow], words[az.currentHigh])
						} else {
							irccon.Privmsgf(channel, "WOW you won, that was the right word %s!", words[foundIndex])
							az = nil
						}
					} else {
						irccon.Privmsgf(channel, "%s is not a valid word. try again", guess)
					}
				}
			}
		}
	})
	err = irccon.Connect(serverssl)
	if err != nil {
		fmt.Printf("Err %s", err)
		return
	}

	irccon.Loop()
}
