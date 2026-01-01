package sidekiq

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// SortedSet is a base type for Sidekiq sorted sets (schedule, retry, dead).
type SortedSet struct {
	client *Client
	name   string
}

// newSortedSet creates a new SortedSet.
func newSortedSet(client *Client, name string) *SortedSet {
	return &SortedSet{
		client: client,
		name:   name,
	}
}

// Name returns the set name.
func (s *SortedSet) Name() string {
	return s.name
}

// Size returns the number of entries in the set.
func (s *SortedSet) Size(ctx context.Context) (int64, error) {
	size, err := s.client.rdb.ZCard(ctx, s.client.key(s.name)).Result()
	if err != nil {
		return 0, wrapRedisErr("get set size", err)
	}
	return size, nil
}

// Clear removes all entries from the set.
// Returns the number of entries deleted.
func (s *SortedSet) Clear(ctx context.Context) (int64, error) {
	key := s.client.key(s.name)
	size, err := s.client.rdb.ZCard(ctx, key).Result()
	if err != nil {
		return 0, wrapRedisErr("clear set", err)
	}
	err = s.client.rdb.Del(ctx, key).Err()
	if err != nil {
		return 0, wrapRedisErr("clear set", err)
	}
	return size, nil
}

// JobSet extends SortedSet with job-specific operations.
type JobSet struct {
	*SortedSet
}

// newJobSet creates a new JobSet.
func newJobSet(client *Client, name string) *JobSet {
	return &JobSet{
		SortedSet: newSortedSet(client, name),
	}
}

// Each iterates over all entries in the set.
// Entries are yielded in score order (oldest first).
func (js *JobSet) Each(ctx context.Context, fn func(*SortedEntry) bool) error {
	const pageSize = 100
	offset := int64(0)

	for {
		entries, err := js.client.rdb.ZRangeWithScores(ctx, js.client.key(js.name), offset, offset+pageSize-1).Result()
		if err != nil {
			return wrapRedisErr("iterate set", err)
		}

		if len(entries) == 0 {
			break
		}

		for _, z := range entries {
			value := z.Member.(string)
			entry, err := newSortedEntry(js, value, z.Score)
			if err != nil {
				continue // Skip malformed entries
			}
			if !fn(entry) {
				return nil
			}
		}

		if int64(len(entries)) < pageSize {
			break
		}
		offset += pageSize
	}

	return nil
}

// EachByScore iterates over entries within the given score range.
func (js *JobSet) EachByScore(ctx context.Context, min, max float64, fn func(*SortedEntry) bool) error {
	const pageSize = 100
	offset := int64(0)

	opt := &redis.ZRangeBy{
		Min:    formatScore(min),
		Max:    formatScore(max),
		Offset: 0,
		Count:  pageSize,
	}

	for {
		opt.Offset = offset
		entries, err := js.client.rdb.ZRangeByScoreWithScores(ctx, js.client.key(js.name), opt).Result()
		if err != nil {
			return wrapRedisErr("iterate set by score", err)
		}

		if len(entries) == 0 {
			break
		}

		for _, z := range entries {
			value := z.Member.(string)
			entry, err := newSortedEntry(js, value, z.Score)
			if err != nil {
				continue
			}
			if !fn(entry) {
				return nil
			}
		}

		if int64(len(entries)) < pageSize {
			break
		}
		offset += pageSize
	}

	return nil
}

// FindJob finds an entry by JID.
// Returns nil if not found.
func (js *JobSet) FindJob(ctx context.Context, jid string) (*SortedEntry, error) {
	var found *SortedEntry
	err := js.Each(ctx, func(entry *SortedEntry) bool {
		if entry.JID() == jid {
			found = entry
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

// Fetch retrieves a specific entry by score and JID.
func (js *JobSet) Fetch(ctx context.Context, score float64, jid string) (*SortedEntry, error) {
	scoreStr := formatScore(score)
	entries, err := js.client.rdb.ZRangeByScoreWithScores(ctx, js.client.key(js.name), &redis.ZRangeBy{
		Min: scoreStr,
		Max: scoreStr,
	}).Result()
	if err != nil {
		return nil, wrapRedisErr("fetch entry", err)
	}

	for _, z := range entries {
		value := z.Member.(string)
		entry, err := newSortedEntry(js, value, z.Score)
		if err != nil {
			continue
		}
		if entry.JID() == jid {
			return entry, nil
		}
	}

	return nil, nil
}

// Schedule adds a job to the set at the given time.
func (js *JobSet) Schedule(ctx context.Context, at time.Time, job map[string]interface{}) error {
	data, err := json.Marshal(job)
	if err != nil {
		return &JobParseError{Err: err}
	}

	err = js.client.rdb.ZAdd(ctx, js.client.key(js.name), redis.Z{
		Score:  float64(at.Unix()),
		Member: string(data),
	}).Err()
	return wrapRedisErr("schedule job", err)
}

// DeleteByValue removes an entry by its exact value.
func (js *JobSet) DeleteByValue(ctx context.Context, value string) (bool, error) {
	removed, err := js.client.rdb.ZRem(ctx, js.client.key(js.name), value).Result()
	if err != nil {
		return false, wrapRedisErr("delete by value", err)
	}
	return removed > 0, nil
}

// DeleteByScore removes all entries with the given score.
func (js *JobSet) DeleteByScore(ctx context.Context, score float64) (int64, error) {
	scoreStr := formatScore(score)
	removed, err := js.client.rdb.ZRemRangeByScore(ctx, js.client.key(js.name), scoreStr, scoreStr).Result()
	if err != nil {
		return 0, wrapRedisErr("delete by score", err)
	}
	return removed, nil
}

// formatScore formats a float64 score for Redis commands.
func formatScore(score float64) string {
	return strconv.FormatFloat(score, 'f', -1, 64)
}
