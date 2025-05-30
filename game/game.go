package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"tcr/models"
	"time"
)

// Game represents the game state
type Game struct {
	Players     [2]*models.Player
	CurrentTurn int
	TroopList   []models.Troop
	TowerList   []models.Tower
	Mutex       sync.Mutex
}

// NewGame creates a new game instance
func NewGame() *Game {
	game := &Game{
		CurrentTurn: 0,
	}
	game.loadData()
	return game
}

// loadData loads towers and troops from JSON files
func (g *Game) loadData() {
	towerData, err := os.ReadFile("../data/towers.json")
	if err != nil {
		fmt.Println("Error reading towers.json:", err)
		os.Exit(1)
	}
	if err := json.Unmarshal(towerData, &g.TowerList); err != nil {
		fmt.Println("Error parsing towers.json:", err)
		os.Exit(1)
	}
	fmt.Printf("Loaded towers: %v\n", g.TowerList)

	troopData, err := os.ReadFile("../data/troops.json")
	if err != nil {
		fmt.Println("Error reading troops.json:", err)
		os.Exit(1)
	}
	if err := json.Unmarshal(troopData, &g.TroopList); err != nil {
		fmt.Println("Error parsing troops.json:", err)
		os.Exit(1)
	}
	fmt.Printf("Loaded troops: %v\n", g.TroopList)
}

// InitializePlayer sets up a player's initial towers, troops, and Mana
func (g *Game) InitializePlayer(player *models.Player) {
	// Clear existing towers and troops
	player.Towers = nil
	player.Troops = nil
	// Assign towers
	for _, tower := range g.TowerList {
		if tower.Type == "King Tower" {
			player.Towers = append(player.Towers, tower)
		}
	}
	for _, tower := range g.TowerList {
		if tower.Type == "Guard Tower" {
			player.Towers = append(player.Towers, tower, tower)
			break // Ensure only two Guard Towers
		}
	}
	// Assign troops with level-based ATK boost
	for _, troop := range g.TroopList {
		t := troop
		atkBoost := float64(player.Level-1) * 0.1
		t.ATK = int(float64(t.ATK) * (1 + atkBoost)) // Sửa lỗi t.tK và ATK int
		player.Troops = append(player.Troops, t)
	}
	// Initialize Mana
	player.Mana = 5 + (player.Level - 1) // Sửa lỗi dấu ngoặc thừa
	player.MaxMana = 10 + (player.Level - 1)
}

// ResetGame resets the game state for a new match
func (g *Game) ResetGame() {
	for _, player := range g.Players {
		wins := player.Wins
		username := player.Username
		password := player.Password
		level := player.Level
		exp := player.EXP
		player.Towers = nil
		player.Troops = nil
		player.Mana = 0
		player.MaxMana = 0
		g.InitializePlayer(player)
		player.Wins = wins
		player.Username = username
		player.Password = password
		player.Level = level
		player.EXP = exp
		fmt.Printf("Reset player %s: Mana=%d/%d, Wins=%d, Level=%d, EXP=%d, Towers=%v, Troops=%v\n",
			player.Username, player.Mana, player.MaxMana, player.Wins, player.Level, player.EXP, player.Towers, player.Troops)
	}
	g.CurrentTurn = 0
}

// Attack processes a player's attack with target choice
func (g *Game) Attack(attacker, defender *models.Player, troopIdx, targetIdx int) string {
	if troopIdx < 0 || troopIdx >= len(attacker.Troops) {
		return "Invalid troop selection"
	}

	troop := attacker.Troops[troopIdx]
	fmt.Printf("Player %s deploying troop: %s (ATK=%d, MANA=%d)\n",
		attacker.Username, troop.Name, troop.ATK, troop.MANA)

	g.Mutex.Lock()
	if attacker.Mana < troop.MANA {
		g.Mutex.Unlock()
		return fmt.Sprintf("Insufficient Mana: Need %d, Have %d", troop.MANA, attacker.Mana)
	}
	attacker.Mana -= troop.MANA
	attacker.EXP += troop.EXP
	requiredEXP := attacker.Level * 100
	levelUpMsg := ""
	for attacker.EXP >= requiredEXP {
		attacker.Level++
		attacker.EXP -= requiredEXP
		requiredEXP = attacker.Level * 100
		g.InitializePlayer(attacker)
		levelUpMsg += fmt.Sprintf("\n%s leveled up to Level %d! Troops ATK +10%%, MaxMana=%d",
			attacker.Username, attacker.Level, attacker.MaxMana)
		fmt.Printf("Player %s leveled up to %d, EXP=%d\n", attacker.Username, attacker.Level, attacker.EXP)
	}
	g.Mutex.Unlock()

	if troop.Name == "Queen" {
		lowestHP := 999999
		var targetTower *models.Tower
		for i, tower := range attacker.Towers {
			if tower.HP > 0 && tower.HP < lowestHP {
				lowestHP = tower.HP
				targetTower = &attacker.Towers[i]
			}
		}
		if targetTower != nil {
			targetTower.HP += 300
			return fmt.Sprintf("Queen healed %s by 300 HP", targetTower.Type) + levelUpMsg
		}
		return "No valid tower to heal" + levelUpMsg
	}

	// Count remaining Guard Towers
	guardTowersAlive := 0
	var lastGuardTower *models.Tower
	for i, tower := range defender.Towers {
		if tower.Type == "Guard Tower" && tower.HP > 0 {
			guardTowersAlive++
			lastGuardTower = &defender.Towers[i]
		}
	}

	var targetTower *models.Tower
	if guardTowersAlive == 2 {
		for i, tower := range defender.Towers {
			if tower.Type == "Guard Tower" && tower.HP > 0 {
				targetTower = &defender.Towers[i]
				break
			}
		}
	} else if guardTowersAlive == 1 {
		if targetIdx == 0 {
			targetTower = lastGuardTower
		} else if targetIdx == 1 {
			for i, tower := range defender.Towers {
				if tower.Type == "King Tower" && tower.HP > 0 {
					targetTower = &defender.Towers[i]
					break
				}
			}
		} else {
			return "Invalid target selection" + levelUpMsg
		}
	} else if guardTowersAlive == 0 {
		if targetIdx == 1 {
			for i, tower := range defender.Towers {
				if tower.Type == "King Tower" && tower.HP > 0 {
					targetTower = &defender.Towers[i]
					break
				}
			}
		} else {
			return "Invalid target selection" + levelUpMsg
		}
	}

	if targetTower == nil {
		return "No valid target tower" + levelUpMsg
	}

	// Thêm logic Critical Damage
	rand.Seed(time.Now().UnixNano())      // Khởi tạo seed để tạo số ngẫu nhiên
	critChance := targetTower.CRIT        // Lấy xác suất CRIT từ tháp mục tiêu
	isCrit := rand.Float64() < critChance // Kiểm tra xem có CRIT hay không
	dmg := troop.ATK                      // Bắt đầu với ATK cơ bản
	if isCrit {
		dmg = int(float64(troop.ATK) * 1.2) // Nhân 1.2 nếu CRIT
	}
	dmg -= targetTower.DEF // Trừ DEF của tháp
	if dmg < 0 {
		dmg = 0
	}
	targetTower.HP -= dmg

	result := fmt.Sprintf("%s dealt %d damage to %s", troop.Name, dmg, targetTower.Type)
	if isCrit {
		result += " (CRITICAL HIT!)" // Hiển thị nếu CRIT xảy ra
	}
	fmt.Printf("Attack result: %s (Target HP=%d)\n", result, targetTower.HP)
	if targetTower.HP <= 0 {
		result += ". " + targetTower.Type + " destroyed!"
		targetTower.HP = 0
	}

	return result + levelUpMsg
}

// RegenerateMana increases a player's Mana
func (g *Game) RegenerateMana(player *models.Player) {
	g.Mutex.Lock()
	if player.Mana < player.MaxMana {
		player.Mana++
		fmt.Printf("Player %s Mana regenerated to %d\n", player.Username, player.Mana)
	}
	g.Mutex.Unlock()
}

// IsGameOver checks if the game has ended
func (g *Game) IsGameOver() (bool, string) {
	for i, player := range g.Players {
		for _, tower := range player.Towers {
			if tower.Type == "King Tower" && tower.HP <= 0 {
				g.Mutex.Lock()
				g.Players[1-i].Wins++
				g.Mutex.Unlock()
				return true, fmt.Sprintf("%s wins! Score: %s=%d (Lv%d), %s=%d (Lv%d)",
					g.Players[1-i].Username, g.Players[0].Username, g.Players[0].Wins, g.Players[0].Level,
					g.Players[1].Username, g.Players[1].Wins, g.Players[1].Level)
			}
		}
	}
	if len(g.Players[0].Troops) == 0 && len(g.Players[1].Troops) == 0 {
		return true, fmt.Sprintf("Draw: Both players ran out of troops. Score: %s=%d (Lv%d), %s=%d (Lv%d)",
			g.Players[0].Username, g.Players[0].Wins, g.Players[0].Level,
			g.Players[1].Username, g.Players[1].Wins, g.Players[1].Level)
	}
	return false, ""
}
