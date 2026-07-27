package main

type LikeTypeData struct {
	Table       string
	CommChannel int
	LikeChannel int
}

var LikeTypes = map[int]LikeTypeData{
	0:  {"gdpses", 0, 0},
	1:  {"texures", 0, 2},
	2:  {"news", 0, 6},
	7:  {"guides", 0, 7},
	8:  {"wikis", 0, 8},
	9:  {"forumPosts", 0, 9},
	11: {"vacans", 0, 11},

	// FIXME: придумать что делать с каналами комментариев тут
	3:  {"comments", 0, 1},  // gdpses
	4:  {"comments", 1, 4},  // texures
	6:  {"comments", 2, 5},  // guides
	5:  {"comments", 3, 4},  // news
	10: {"comments", 4, 10}, // forums
	12: {"comments", 5, 12}, // vacs
}

var CommLikeTypes = map[int]LikeTypeData{
	0: {"comments", 0, 1},  // gdpses
	1: {"comments", 1, 4},  // texures
	2: {"comments", 2, 5},  // guides
	3: {"comments", 3, 4},  // news
	4: {"comments", 4, 10}, // forums
	5: {"comments", 5, 12}, // vacs
}

// мне лень рефакторить бекенд на внятные константы, пусть магия останется
var channelPrefixes = map[int]string{
	0: "c",
	1: "s",
	2: "p",
	3: "t",
}

func ChannelIdsToString(id int) string {
	prefix, ok := channelPrefixes[id]
	if !ok {
		return "x"
	}
	return prefix
}
