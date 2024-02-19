package config

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Piszmog/cloudconfigclient/v2"
)

type Data struct {
	InstanceId int64  `json:"user_id" yaml:"user_id"`
	Token      string `json:"api_token_id" yaml:"api_token_id"`
	Link1      string `json:"link_1" yaml:"link_1"`
	Link2      string `json:"link_2" yaml:"link_2"`
	Link3      string `json:"link_3" yaml:"link_3"`
	Link4      string `json:"link_4" yaml:"link_4"`
}

type CloudConfig struct {
	client *cloudconfigclient.Client
	data   map[string]Data
}

func NewCloudConfig() *CloudConfig {
	cc := CloudConfig{
		data: make(map[string]Data),
	}

	err := cc.Init()
	if err != nil {
		log.Fatalln(err)
	}

	return &cc
}

func (cc *CloudConfig) Init() error {
	client, err := cloudconfigclient.New(cloudconfigclient.Local(&http.Client{}, os.Getenv("SPRING_CLOUD_CONFIG_URI")))
	if err != nil {
		return err
	}

	cc.client = client

	return nil
}

func (cc *CloudConfig) Load(app string, profiles ...string) error {
	var data Data

	if len(profiles) == 0 {
		return errors.New("no profiles provided")
	}

	key := strings.Join([]string{app, profiles[0]}, "-")

	if _, ok := cc.data[key]; !ok {
		configuration, err := cc.client.GetConfiguration(app, profiles...)
		if err != nil {
			return err
		}

		err = unmarshal(&configuration, &data)
		if err != nil {
			return err
		}

		cc.data[key] = data
	}

	return nil
}

func (cc *CloudConfig) Get(name string) (*Data, error) {
	if data, ok := cc.data[name]; ok {
		return &data, nil
	}

	return nil, errors.New("no config found")
}

func reverseAndCopy(arr []cloudconfigclient.PropertySource) []cloudconfigclient.PropertySource {
	reversed := make([]cloudconfigclient.PropertySource, len(arr))
	for i := 0; i < len(arr); i++ {
		reversed[i] = arr[len(arr)-1-i]
	}
	return reversed
}

func unmarshal(s *cloudconfigclient.Source, v interface{}) error {
	// covert to a map[string]interface{} so we can convert to the target type
	obj, err := toJson(reverseAndCopy(s.PropertySources))
	if err != nil {
		return err
	}
	// convert to bytes
	b, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	// now we can get to our target type
	return json.Unmarshal(b, v)
}

var sliceRegex = regexp.MustCompile(`(.*)\[(\d+)]`)

func toJson(propertySources []cloudconfigclient.PropertySource) (map[string]interface{}, error) {
	// get ready for a wild ride...
	output := map[string]interface{}{}
	// save the root, so we can get back there when we walk the tree
	root := output
	_ = root
	for _, propertySource := range propertySources {
		for k, v := range propertySource.Source {
			keys := strings.Split(k, ".")
			for i, key := range keys {
				// determine if we are detailing with a slice - e.g. foo[0] or bar[0]
				matches := sliceRegex.FindStringSubmatch(key)
				if matches != nil {
					actualKey := matches[1]
					if _, ok := output[actualKey]; !ok {
						output[actualKey] = []interface{}{}
					}
					if len(keys)-1 == i {
						// the value go straight into the slice, we don't have any slice of objects
						output[actualKey] = append(output[actualKey].([]interface{}), v)
						output = root
					} else {
						// ugh... we have a slice of objects
						index, err := strconv.Atoi(matches[2])
						if err != nil {
							return nil, err
						}
						var obj map[string]interface{}
						slice := output[actualKey].([]interface{})
						// determine if the index we are walking exists yet in the slice we have built up
						if len(slice) > index {
							obj = slice[index].(map[string]interface{})
							if obj == nil {
								obj = map[string]interface{}{}
							}
						} else {
							// the index does not exist, so we need to create it
							for j := len(slice); j <= index; j++ {
								output[actualKey] = append(output[actualKey].([]interface{}), map[string]interface{}{})
							}
							obj = output[actualKey].([]interface{})[index].(map[string]interface{})
						}
						output = obj
					}
				} else if len(keys)-1 == i {
					// the value go straight into the key
					output[key] = v
					output = root
				} else {
					// need to create a nested object
					if _, ok := output[key]; !ok {
						output[key] = map[string]interface{}{}
					}
					output = output[key].(map[string]interface{})
				}
			}
		}
	}
	return output, nil
}
