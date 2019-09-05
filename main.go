package main

import (
	"crypto/tls"
	"fmt"
	"github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
	"net/http"
	"os"
	"reflect"
)

var log = logrus.New()
var Version string

func main() {
	var (
		app = kingpin.New("cf-ddns", "Cloudflare DynDNS Updater").Version(Version)

		ipAddress = app.Flag("ip-address", "Skip resolving external IP and use provided IP").String()
		noVerify  = app.Flag("no-verify", "Don't verify ssl certificates").Bool()

		cfEmail  = app.Flag("cf-email", "Cloudflare Email").Required().String()
		cfApiKey = app.Flag("cf-api-key", "Cloudflare API key").Required().String()
		cfZoneId = app.Flag("cf-zone-id", "Cloudflare Zone ID").Required().String()

		hostnames = app.Arg("hostnames", "Hostnames to update").Required().Strings()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	var ip IPService
	var dns *CFDNSUpdater
	var store Storage
	var err error

	if *ipAddress != "" {
		ip = &FakeIPService{
			fakeIp: net.ParseIP(*ipAddress),
		}
	} else {
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: *noVerify},
			},
		}
		ip = &IpifyIPService{HttpClient: httpClient}
	}

	stored_ip, err := store.GetIP()
	if err != nil {
		fmt.Printf("No IP stored yet\n")
	}

	res, err := ip.GetExternalIP()
	if err != nil {
		log.Panic(err)
	}

	log.Debugf("Stored IP: %s, External IP: %s", stored_ip, res)

	if stored_ip != nil {
		if !reflect.DeepEqual(stored_ip, res) {
			err = store.PutIP(res)
			if err != nil {
				log.Panic(err)
			}

			if dns, err = NewCFDNSUpdater(*cfZoneId, *cfApiKey, *cfEmail, log.WithField("component", "cf-dns-updater")); err != nil {
				log.Panic(err)
			}

			for _, hostname := range *hostnames {
				err := dns.UpdateRecordA(hostname, res)
				if err != nil {
					log.Panic(err)
				}
			}
		}
	} else {
		err = store.PutIP(res)
		if err != nil {
			log.Panic(err)
		}
	}
}
