package sidekiq

// ScheduledSet represents Sidekiq's scheduled job set.
// Jobs here are waiting to be processed at a future time.
type ScheduledSet struct {
	*JobSet
}

// NewScheduledSet creates a new ScheduledSet.
func NewScheduledSet(client *Client) *ScheduledSet {
	return &ScheduledSet{
		JobSet: newJobSet(client, keySchedule),
	}
}
