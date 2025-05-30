package models

type Player struct {
	Username    string  `json:"username"`
	Password    string  `json:"password"`
	Towers      []Tower `json:"towers"`
	Troops      []Troop `json:"troops"`
	ActiveTroop *Troop  `json:"active_troop"`
	Mana        int     `json:"mana"`
	MaxMana     int     `json:"max_mana"`
	Wins        int     `json:"wins"`
	Level       int     `json:"level"` // Player level (starts at 1)
	EXP         int     `json:"exp"`   // Current EXP
}
