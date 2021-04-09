package rmqhttp

// Just what needs to be used, nothing fancy.
type rate struct {
	Rate float32
}

type messageStats struct {
	AckDetails     rate `json:"ack_details"`
	PublishDetails rate `json:"publish_details"`
}

type rmqStats struct {
	Messages     int
	MessageStats messageStats `json:"message_stats"`
}
