package relaypool

const (
	PublishStatusSent      = 0
	PublishStatusFailed    = -1
	PublishStatusSucceeded = 1
)

type PublishStatus struct {
	Relay  string
	Status int
}
