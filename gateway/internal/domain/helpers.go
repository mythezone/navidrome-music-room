package domain

import (
	"crypto/rand"
	"math/big"
	"slices"
	"strings"
	"time"
)

func EqualUsername(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func EffectivePosition(state PlaybackState, now time.Time) float64 {
	position := state.PositionSeconds
	if state.Status == PlaybackPlaying && state.AnchorServerTime != nil {
		position += now.Sub(*state.AnchorServerTime).Seconds()
	}
	if position < 0 {
		return 0
	}
	if state.CurrentTrack != nil && state.CurrentTrack.DurationSeconds > 0 && position > state.CurrentTrack.DurationSeconds {
		return state.CurrentTrack.DurationSeconds
	}
	return position
}

func SelectNext(entries []QueueEntry, mode, lastContributor string) *QueueEntry {
	if len(entries) == 0 {
		return nil
	}
	ordered := slices.Clone(entries)
	slices.SortFunc(ordered, func(a, b QueueEntry) int { return a.Position - b.Position })
	contributors := make([]string, 0, len(ordered))
	seen := map[string]struct{}{}
	for _, entry := range ordered {
		key := strings.ToLower(entry.Contributor)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			contributors = append(contributors, entry.Contributor)
		}
	}
	eligible := contributors
	if len(contributors) > 1 {
		eligible = nil
		for _, contributor := range contributors {
			if !EqualUsername(contributor, lastContributor) {
				eligible = append(eligible, contributor)
			}
		}
	}
	if mode == QueueFIFO {
		for _, entry := range ordered {
			for _, contributor := range eligible {
				if EqualUsername(entry.Contributor, contributor) {
					copy := entry
					return &copy
				}
			}
		}
	}
	contributor := eligible[secureIndex(len(eligible))]
	var candidates []QueueEntry
	for _, entry := range ordered {
		if EqualUsername(entry.Contributor, contributor) {
			candidates = append(candidates, entry)
		}
	}
	selected := candidates[secureIndex(len(candidates))]
	return &selected
}

func secureIndex(length int) int {
	if length <= 1 {
		return 0
	}
	value, err := rand.Int(rand.Reader, big.NewInt(int64(length)))
	if err != nil {
		return 0
	}
	return int(value.Int64())
}

func ContainsFolder(allowed []int, target int) bool {
	return slices.Contains(allowed, target)
}

func ContainsAllFolders(allowed, required []int) bool {
	for _, folder := range required {
		if !ContainsFolder(allowed, folder) {
			return false
		}
	}
	return true
}
