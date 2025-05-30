package main

import (
	"encoding/json"
	"fmt"
	"net"
	"tcr/game"
	"tcr/models"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	fmt.Print("Enter username: ")
	var username, password string
	fmt.Scanln(&username)
	fmt.Print("Enter password: ")
	fmt.Scanln(&password)

	if err := json.NewEncoder(conn).Encode(struct {
		Username string
		Password string
	}{username, password}); err != nil {
		fmt.Println("Error sending authentication:", err)
		return
	}

	var player models.Player
	if err := json.NewDecoder(conn).Decode(&player); err != nil {
		fmt.Println("Error receiving player data:", err)
		return
	}
	fmt.Printf("Player: Username=%s, Mana=%d/%d, Wins=%d, Level=%d, EXP=%d, Towers=%v, Troops=%v\n",
		player.Username, player.Mana, player.MaxMana, player.Wins, player.Level, player.EXP, player.Towers, player.Troops)

	for {
		var gameState game.Game
		if err := json.NewDecoder(conn).Decode(&gameState); err != nil {
			fmt.Println("Error receiving game state:", err)
			return
		}

		// Determine player index (0 or 1)
		playerIdx := 0
		if gameState.Players[1].Username == player.Username {
			playerIdx = 1
		}
		enemyIdx := 1 - playerIdx

		fmt.Printf("Score: You=%d (Lv%d), Enemy=%d (Lv%d)\n",
			gameState.Players[playerIdx].Wins, gameState.Players[playerIdx].Level,
			gameState.Players[enemyIdx].Wins, gameState.Players[enemyIdx].Level)
		fmt.Printf("Your Level: %d, EXP: %d/%d\n",
			gameState.Players[playerIdx].Level, gameState.Players[playerIdx].EXP, gameState.Players[playerIdx].Level*100)
		fmt.Printf("Your Mana: %d/%d, Enemy Mana: %d/%d\n",
			gameState.Players[playerIdx].Mana, player.MaxMana,
			gameState.Players[enemyIdx].Mana, gameState.Players[enemyIdx].MaxMana)
		fmt.Println("Your Towers:")
		for _, tower := range gameState.Players[playerIdx].Towers {
			fmt.Printf("%s: HP=%d\n", tower.Type, tower.HP)
		}
		fmt.Println("Enemy Towers:")
		for _, tower := range gameState.Players[enemyIdx].Towers {
			fmt.Printf("%s: HP=%d\n", tower.Type, tower.HP)
		}
		fmt.Println("Your Troops:")
		for i, troop := range gameState.Players[playerIdx].Troops {
			fmt.Printf("%d: %s (HP=%d, ATK=%d, MANA=%d, EXP=%d)\n",
				i, troop.Name, troop.HP, troop.ATK, troop.MANA, troop.EXP)
		}

		fmt.Print("Select troop index to deploy: ")
		var troopIdx int
		fmt.Scanln(&troopIdx)

		// Count enemy's live Guard Towers
		guardTowersAlive := 0
		for _, tower := range gameState.Players[enemyIdx].Towers {
			if tower.Type == "Guard Tower" && tower.HP > 0 {
				guardTowersAlive++
			}
		}

		targetIdx := 0
		if guardTowersAlive <= 1 {
			fmt.Printf("Choose target: 0=Guard Tower (%d available), 1=King Tower\n", guardTowersAlive)
			fmt.Scanln(&targetIdx)
		}

		if err := json.NewEncoder(conn).Encode(struct {
			TroopIdx  int
			TargetIdx int
		}{troopIdx, targetIdx}); err != nil {
			fmt.Println("Error sending action:", err)
			return
		}

		var result string
		if err := json.NewDecoder(conn).Decode(&result); err != nil {
			fmt.Println("Error receiving attack result:", err)
			return
		}
		fmt.Println("Attack Result:", result)

		var gameOverResult string
		if err := json.NewDecoder(conn).Decode(&gameOverResult); err != nil {
			fmt.Println("Error receiving game over result:", err)
			return
		}
		fmt.Printf("Game Over Result: %q\n", gameOverResult)
		if gameOverResult != "" {
			fmt.Println("Match Over:", gameOverResult)
			continue
		}
	}
}
