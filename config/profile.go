package config

import (
	"strings"
)

type Profile struct {
	Env    string
	Pool   string
	Server string
}

func ParseProfile(activeProfile string) *Profile {
	profile := &Profile{}

	splits := strings.Split(activeProfile, ",")

	if len(splits) == 1 {
		if len(activeProfile) != 7 {
			profile.Pool = activeProfile
		} else {
			profile.Pool = activeProfile[:4]
			profile.Server = activeProfile[4:]
		}
	} else if len(splits) == 2 {
		profile.Env = splits[0]

		if len(splits[1]) == 3 {
			profile.Server = splits[1]
		} else if len(splits[1]) == 4 {
			profile.Pool = splits[1]
		} else {
			profile.Pool = splits[1][:4]
			profile.Server = splits[1][4:]
		}
	}

	return profile
}
