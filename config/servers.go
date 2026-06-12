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
	Scania: {
		"35.163.4.248",   // Channel 1
		"54.69.121.239",  // Channel 2
		"52.27.135.94",   // Channel 3
		"34.218.55.122",  // Channel 4
		"54.213.105.170", // Channel 5
		"52.37.131.173",  // Channel 6
		"52.38.110.221",  // Channel 7
		"50.112.158.189", // Channel 8
		"34.215.85.101",  // Channel 9
		"54.191.76.216",  // Channel 10
		"54.191.254.95",  // Channel 11
		"50.112.211.236", // Channel 12
		"35.165.21.160",  // Channel 13
		"34.211.249.74",  // Channel 14
		"52.43.74.100",   // Channel 15
		"34.209.206.177", // Channel 16
		"34.214.52.19",   // Channel 17
		"54.189.248.141", // Channel 18
		"34.208.240.38",  // Channel 19
		"54.245.14.209",  // Channel 20
		"52.26.44.15",    // Channel 21
		"52.88.199.249",  // Channel 22
		"54.71.159.23",   // Channel 23
		"54.200.197.85",  // Channel 24
		"52.24.108.169",  // Channel 25
		"52.32.48.160",   // Channel 26
		"52.27.243.250",  // Channel 27
		"54.203.90.46",   // Channel 28
		"54.148.240.123", // Channel 29
		"35.164.217.126", // Channel 30
	},
	Bera: {
		"54.186.151.49",  // Channel 1
		"54.214.207.253", // Channel 2
		"34.214.214.251", // Channel 3
		"35.165.105.161", // Channel 4
		"35.167.16.143",  // Channel 5
		"52.40.39.138",   // Channel 6
		"54.68.47.217",   // Channel 7
		"52.35.241.179",  // Channel 8
		"34.218.68.31",   // Channel 9
		"52.43.9.29",     // Channel 10
		"54.213.64.154",  // Channel 11
		"52.25.121.0",    // Channel 12
		"54.148.5.57",    // Channel 13
		"35.161.154.148", // Channel 14
		"54.203.140.45",  // Channel 15
		"35.163.184.1",   // Channel 16
		"34.218.100.191", // Channel 17
		"52.38.89.169",   // Channel 18
		"52.88.17.178",   // Channel 19
		"52.27.189.124",  // Channel 20
		"44.234.162.131", // Channel 21
		"44.234.161.98",  // Channel 22
		"44.234.161.51",  // Channel 23
		"44.234.161.75",  // Channel 24
		"44.234.162.103", // Channel 25
		"44.234.160.97",  // Channel 26
		"44.234.161.91",  // Channel 27
		"44.234.161.240", // Channel 28
		"44.234.160.81",  // Channel 29
		"44.234.162.143", // Channel 30
	},
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
	Hyperion: {
		"44.234.161.190", // Channel 1
		"44.234.161.196", // Channel 2
		"44.234.162.144", // Channel 3
		"44.234.162.237", // Channel 4
		"44.234.164.194", // Channel 5
		"44.234.164.238", // Channel 6
		"44.234.167.164", // Channel 7
		"44.234.167.218", // Channel 8
		"44.234.168.226", // Channel 9
		"44.234.170.140", // Channel 10
		"44.234.171.210", // Channel 11
		"44.234.175.199", // Channel 12
		"44.234.176.213", // Channel 13
		"44.234.179.113", // Channel 14
		"44.234.179.122", // Channel 15
		"44.234.180.145", // Channel 16
		"44.234.181.127", // Channel 17
		"44.234.181.165", // Channel 18
		"44.234.182.249", // Channel 19
		"44.234.184.180", // Channel 20
		"44.234.180.72",  // Channel 21
		"44.234.159.217", // Channel 22
		"44.234.184.107", // Channel 23
		"44.234.165.250", // Channel 24
		"44.234.165.130", // Channel 25
		"44.234.174.53",  // Channel 26
		"44.234.183.141", // Channel 27
		"44.234.78.21",   // Channel 28
		"44.234.169.212", // Channel 29
		"44.234.166.166", // Channel 30
	},
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
