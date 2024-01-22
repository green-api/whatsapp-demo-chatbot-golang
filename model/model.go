package model

type PollMessage struct {
	MessageData struct {
		PollMessageData struct {
			Votes []Vote `json:"votes"`
		} `json:"pollMessageData"`
	} `json:"messageData"`
}

type Vote struct {
	OptionName   string
	OptionVoters []string
}
