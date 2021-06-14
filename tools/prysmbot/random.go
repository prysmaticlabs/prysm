package main

import (
	"fmt"
	"math/rand"
)

func isRandomCommand(command string) bool {
	for _, randomCommand := range randomCommandGroup.commands {
		if command == randomCommand.command || command == randomCommand.shorthand {
			return true
		}
	}
	return false
}

func getRandomResult(command string) string {
	switch command {
	case randomFood.command, randomFood.shorthand:
		return fmt.Sprintf(randomFood.responseText, foods[rand.Int()%len(foods)])
	case randomRestaurant.command, randomRestaurant.shorthand:
		return fmt.Sprintf(randomFood.responseText, restaurants[rand.Int()%len(restaurants)])
	default:
		return ""
	}
}
