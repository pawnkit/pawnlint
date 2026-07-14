package project

import "strings"

func MatchGlob(pattern, path string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	patSeg := splitSegments(pattern)
	pathSeg := splitSegments(path)
	return matchSegments(patSeg, pathSeg)
}

func splitSegments(s string) []string {
	s = strings.ReplaceAll(s, "\\", "/")
	return strings.Split(s, "/")
}

func matchSegments(pat, path []string) bool {
	n, m := len(pat), len(path)
	dp := make([][]bool, n+1)
	for i := range dp {
		dp[i] = make([]bool, m+1)
	}
	dp[n][m] = true
	for i := n - 1; i >= 0; i-- {
		for j := m; j >= 0; j-- {
			if pat[i] == "**" {
				dp[i][j] = dp[i+1][j] || (j < m && dp[i][j+1])
				continue
			}
			dp[i][j] = j < m && matchSeg(pat[i], path[j]) && dp[i+1][j+1]
		}
	}
	return dp[0][0]
}

func matchSeg(pattern, seg string) bool {
	if pattern == "*" {
		return true
	}
	return wildcardMatch(pattern, seg)
}

func wildcardMatch(pattern, s string) bool {
	pi, si := 0, 0
	star := -1
	ssave := 0
	for si < len(s) {
		if pi < len(pattern) && (pattern[pi] == s[si] || pattern[pi] == '?') {
			pi++
			si++
			continue
		}
		if pi < len(pattern) && pattern[pi] == '*' {
			star = pi
			ssave = si
			pi++
			continue
		}
		if star != -1 {
			pi = star + 1
			si = ssave + 1
			ssave = si
			continue
		}
		return false
	}
	for pi < len(pattern) && pattern[pi] == '*' {
		pi++
	}
	return pi == len(pattern)
}
