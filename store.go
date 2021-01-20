package main

import (
	"io/ioutil"
)

type Store interface {
	GetIP() (string, error)
	PutIP(ip string) error
}

type Storage struct{}

var StorePath = "/tmp/cf-ddns-v2.bat"

func (s *Storage) GetIP() (string, error) {
	dat, err := ioutil.ReadFile(StorePath)
	if err != nil {
		return "", err
	}
	return string(dat), nil
}

func (s *Storage) PutIP(ip string) error {
	err := ioutil.WriteFile(StorePath, []byte(ip), 0644)
	return err
}
