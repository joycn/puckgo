package datasource

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// AccessListFromFile get urlactions from path
func AccessListFromFile(path string) (*AccessList, error) {
	info := new(accessListConfig)
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, info)

	if err != nil {
		return nil, err
	}

	return newAccessList(info)
}
