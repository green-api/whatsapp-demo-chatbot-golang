package util

func ContainString(optionVotes []string, targetWid string) bool {
	for _, voter := range optionVotes {
		if voter == targetWid {
			return true
		}
	}
	return false
}
