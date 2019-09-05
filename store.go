package main

import (
	"io/ioutil"
	"net"
)

type Store interface {
	GetIP() (net.IP, error)
	PutIP(ip net.IP) error
}

type Storage struct{}

var StorePath = "/tmp/cf-ddns.bat"

func (s *Storage) GetIP() (net.IP, error) {
	dat, err := ioutil.ReadFile(StorePath)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func (s *Storage) PutIP(ip net.IP) error {
	err := ioutil.WriteFile(StorePath, ip, 0644)
	return err
}
