package main

import (
	"crypto/tls"
	"fmt"
	"github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/cloudflare/cloudflare-go"
	"net"
	"net/http"
	"os"
)

var log = logrus.New()
var Version string

func registerEndpoint(api cloudflare.API, zone string, subdomain string, ip_address string) {

	zoneID, err := api.ZoneIDByName(zone)
	if err != nil {
		log.Fatal(err)
	}
	endpoint := fmt.Sprintf("%s.%s", subdomain, zone)

	record := cloudflare.DNSRecord{
		Type: "A",
		Name: subdomain,
		Content: ip_address,
	}
	log.Debugf("Registering %s -> %s", endpoint, ip_address)
	_, err = api.CreateDNSRecord(zoneID, record)
	if err != nil {
		log.Error(err)
	}
}

func updateEndpoint(api cloudflare.API, zone string, subdomain string, ip_address string) {

	zoneID, err := api.ZoneIDByName(zone)
	if err != nil {
	    log.Fatal(err)
	}
	endpoint := fmt.Sprintf("%s.%s", subdomain, zone)

	vpn := cloudflare.DNSRecord{Name: endpoint}
	log.Debugf("Looking up %s", endpoint)
	recs, err := api.DNSRecords(zoneID, vpn)
	if err != nil {
		log.Fatal(err)
	}

	record := cloudflare.DNSRecord{
		Type: "A",
		Name: subdomain,
		Content: ip_address,
	}
	for _, r := range recs {
		log.Debugf("Updating %s -> %s", r.Name, ip_address)
		err := api.UpdateDNSRecord(zoneID, r.ID, record)
		if err != nil {
			log.Error(err)
		}
	}
}

func main() {
	var (
		app = kingpin.New("cf-ddns", "Cloudflare DynDNS Updater").Version(Version)

		ipAddress = app.Flag("ip-address", "Skip resolving external IP and use provided IP").String()
		noVerify  = app.Flag("no-verify", "Don't verify ssl certificates").Bool()

		cfEmail  = app.Flag("cf-email", "Cloudflare Email").Required().String()
		cfApiKey = app.Flag("cf-api-key", "Cloudflare API key").Required().String()
		cfZoneName = app.Flag("cf-zone-name", "Cloudflare Zone Name").Required().String()

		hostname = app.Arg("hostname", "Hostnames to update").Required().String()
		logLevel = app.Flag("loglevel", "Log level").String()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	var ip IPService
	var store Storage
	var err error

	var endpoint = fmt.Sprintf("%s.%s", *hostname, *cfZoneName)

	ll, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		ll = logrus.InfoLevel
	}
	log.SetLevel(ll)

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
		log.Info("No IP stored yet")
	}

	external_ip, err := ip.GetExternalIP()
	if err != nil {
		log.Error(err)
	}
	log.Debugf("Stored IP: %s, External IP: %s", stored_ip, external_ip.String())

	entries, err := net.LookupHost(endpoint)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("%s resolves to %s", endpoint, entries)

	api, err := cloudflare.New(*cfApiKey, *cfEmail)
	if err != nil {
		log.Fatal(err)
	}

	if len(entries) == 0 {
		registerEndpoint(*api, *cfZoneName, *hostname, external_ip.String())
	} else if len(entries) == 1 {
		if entries[0] != external_ip.String() {
			updateEndpoint(*api, *cfZoneName, *hostname, external_ip.String())
		}
	} else {
		log.Fatalf("More than one entry for %s (%d)", endpoint, len(entries))
	}

	if stored_ip != "" {
		if stored_ip != external_ip.String() {
			updateEndpoint(*api, *cfZoneName, *hostname, external_ip.String())

			err = store.PutIP(external_ip.String())
			if err != nil {
				log.Error(err)
			}
		}
	} else {
		updateEndpoint(*api, *cfZoneName, *hostname, external_ip.String())

		err = store.PutIP(external_ip.String())
		if err != nil {
			log.Error(err)
		}
	}
}
