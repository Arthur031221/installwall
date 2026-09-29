package rules

// EditDistance computes the distance between a and b allowing insertion,
// deletion, substitution, and transposition of two adjacent characters
// (the "optimal string alignment" variant of Damerau-Levenshtein).
//
// Transposition has to count as one edit here, not two: the single most
// common typosquat pattern is two adjacent letters swapped, for example
// "reqeusts" for "requests". Under plain Levenshtein that is distance 2
// (two substitutions) and would slip past a "distance 1" typosquat
// check entirely.
//
// The result is capped at cap+1: once the true distance is known to
// exceed cap, the function may return cap+1 instead of computing the
// exact value.
func EditDistance(a, b string, cap int) int {
	ra, rb := []rune(a), []rune(b)
	n, m := len(ra), len(rb)
	if diff := n - m; diff > cap || -diff > cap {
		return cap + 1
	}

	d := make([][]int, n+1)
	for i := range d {
		d[i] = make([]int, m+1)
	}
	for i := 0; i <= n; i++ {
		d[i][0] = i
	}
	for j := 0; j <= m; j++ {
		d[0][j] = j
	}

	for i := 1; i <= n; i++ {
		rowMin := d[i][0]
		for j := 1; j <= m; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			best := d[i-1][j] + 1             // deletion
			if v := d[i][j-1] + 1; v < best { // insertion
				best = v
			}
			if v := d[i-1][j-1] + cost; v < best { // substitution
				best = v
			}
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				if v := d[i-2][j-2] + 1; v < best { // transposition
					best = v
				}
			}
			d[i][j] = best
			if best < rowMin {
				rowMin = best
			}
		}
		if rowMin > cap {
			return cap + 1
		}
	}
	return d[n][m]
}
