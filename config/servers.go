// Package config holds the static list of game worlds and the channel IP
// addresses to ping.
package config

import "strings"

// World identifies a MapleStory world.
type World string

const (
	Scania   World = "Scania"
	Bera     World = "Bera"
	Kronos   World = "Kronos"
	Hyperion World = "Hyperion"
)

// Servers maps each world to its channel IP addresses. The slice index
// determines the channel number: index 0 is channel 1, and so on. Empty
// worlds are configured but have no channels yet.
var Servers = map[World][]string{
	Scania: {},
	Bera:   {},
	Kronos: {
		"35.155.204.207", // Channel 1
		"52.26.82.74",    // Channel 2
		"34.217.205.66",  // Channel 3
		"35.161.183.101", // Channel 4
		"54.218.157.183", // Channel 5
		"52.25.78.39",    // Channel 6
		"54.68.160.34",   // Channel 7
		"34.218.141.142", // Channel 8
		"52.33.249.126",  // Channel 9
		"54.148.170.23",  // Channel 10
		"54.201.184.26",  // Channel 11
		"54.191.142.56",  // Channel 12
		"52.13.185.207",  // Channel 13
		"34.215.228.37",  // Channel 14
		"54.187.177.143", // Channel 15
		"54.203.83.148",  // Channel 16
		"54.148.188.235", // Channel 17
		"52.43.83.76",    // Channel 18
		"54.69.114.137",  // Channel 19
		"54.148.137.49",  // Channel 20
		"54.212.109.33",  // Channel 21
		"44.230.255.51",  // Channel 22
		"100.20.116.83",  // Channel 23
		"54.188.84.22",   // Channel 24
		"34.215.170.50",  // Channel 25
		"54.184.162.28",  // Channel 26
		"54.185.209.29",  // Channel 27
		"52.12.53.225",   // Channel 28
		"54.189.33.238",  // Channel 29
		"54.188.84.238",  // Channel 30
		"44.234.162.14",  // Channel 31
		"44.234.162.13",  // Channel 32
		"44.234.161.92",  // Channel 33
		"44.234.161.48",  // Channel 34
		"44.234.160.137", // Channel 35
		"44.234.161.28",  // Channel 36
		"44.234.162.100", // Channel 37
		"44.234.161.69",  // Channel 38
		"44.234.162.145", // Channel 39
		"44.234.162.130", // Channel 40
	},
	Hyperion: {},
}

// WorldOrder fixes the iteration order for endpoints and the ping cycle,
// since map iteration order is randomized.
var WorldOrder = []World{Scania, Bera, Kronos, Hyperion}

// TryGetWorld looks up a world by name, case-insensitively.
func TryGetWorld(name string) (World, bool) {
	for _, w := range WorldOrder {
		if strings.EqualFold(string(w), name) {
			return w, true
		}
	}
	return "", false
}

// ChannelCount returns how many channels a world has configured.
func ChannelCount(w World) int {
	return len(Servers[w])
}
