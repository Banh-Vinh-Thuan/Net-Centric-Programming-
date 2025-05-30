package main

import (
	"encoding/json"
	"fmt"
	"net"
	"tcr/game"
	"tcr/models"
	"time"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server started on :8080")

	gameInstance := game.NewGame()
	clients := make([]net.Conn, 0)

	for len(clients) < 2 {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		clients = append(clients, conn)
		fmt.Println("Player connected:", conn.RemoteAddr().String())
	}

	// Initialize players
	for i, conn := range clients {
		player := &models.Player{Wins: 0, Level: 1, EXP: 0}
		decoder := json.NewDecoder(conn)
		encoder := json.NewEncoder(conn)
		var auth struct {
			Username string
			Password string
		}
		if err := decoder.Decode(&auth); err != nil {
			fmt.Println("Error decoding auth for player", i, ":", err)
			conn.Close()
			continue
		}
		player.Username = auth.Username
		player.Password = auth.Password
		gameInstance.Players[i] = player
		gameInstance.InitializePlayer(player)
		fmt.Printf("Player %d initialized: Username=%s, Mana=%d/%d, Wins=%d, Level=%d, EXP=%d, Towers=%v, Troops=%v\n",
			i, player.Username, player.Mana, player.MaxMana, player.Wins, player.Level, player.EXP, player.Towers, player.Troops)
		if err := encoder.Encode(player); err != nil {
			fmt.Println("Error sending player data to player", i, ":", err)
			conn.Close()
			continue
		}

		go func(p *models.Player) {
			ticker := time.NewTicker(2 * time.Second) // Tăng 1 Mana mỗi 2 giây
			defer ticker.Stop()
			for range ticker.C {
				gameInstance.RegenerateMana(p)
			}
		}(player)
	}

	for {
		for i, player := range gameInstance.Players {
			conn := clients[i]
			encoder := json.NewEncoder(conn)
			decoder := json.NewDecoder(conn)

			fmt.Printf("Sending game state to player %d (%s): Your Mana=%d/%d, Enemy Mana=%d/%d, Score: %s=%d (Lv%d), %s=%d (Lv%d)\n",
				i, player.Username, player.Mana, player.MaxMana, gameInstance.Players[1-i].Mana, gameInstance.Players[1-i].MaxMana,
				player.Username, player.Wins, player.Level, gameInstance.Players[1-i].Username, gameInstance.Players[1-i].Wins, gameInstance.Players[1-i].Level)
			if err := encoder.Encode(gameInstance); err != nil {
				fmt.Println("Error sending game state to player", i, ":", err)
				for _, c := range clients {
					c.Close()
				}
				return
			}

			var action struct {
				TroopIdx  int
				TargetIdx int
			}
			if err := decoder.Decode(&action); err != nil {
				fmt.Println("Error receiving action from player", i, ":", err)
				for _, c := range clients {
					c.Close()
				}
				return
			}
			fmt.Printf("Player %d (%s) selected troop=%d, target=%d\n", i, player.Username, action.TroopIdx, action.TargetIdx)

			result := gameInstance.Attack(player, gameInstance.Players[1-i], action.TroopIdx, action.TargetIdx)
			fmt.Printf("Attack result for player %d: %s\n", i, result)
			if err := encoder.Encode(result); err != nil {
				fmt.Println("Error sending attack result to player", i, ":", err)
				for _, c := range clients {
					c.Close()
				}
				return
			}

			over, gameOverResult := gameInstance.IsGameOver()
			fmt.Printf("Game over check for player %d: over=%v, result=%s\n", i, over, gameOverResult)
			if err := encoder.Encode(gameOverResult); err != nil {
				fmt.Println("Error sending game over result to player", i, ":", err)
				for _, c := range clients {
					c.Close()
				}
				return
			}

			if over {
				for j, c := range clients {
					if err := json.NewEncoder(c).Encode(gameOverResult); err != nil {
						fmt.Println("Error sending game over result to player", j, ":", err)
					}
				}
				fmt.Println("Match over, resetting for new match...")
				gameInstance.ResetGame()
				for j, c := range clients {
					if err := json.NewEncoder(c).Encode(gameInstance); err != nil {
						fmt.Println("Error sending new game state to player", j, ":", err)
						for _, c := range clients {
							c.Close()
						}
						return
					}
				}
				continue
			}
		}
	}
}
