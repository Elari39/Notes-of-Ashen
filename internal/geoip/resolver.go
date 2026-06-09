package geoip

import (
	"net"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

const unknown = "Unknown"

type Location struct {
	CountryCode string
	CountryName string
	RegionName  string
	CityName    string
}

type Resolver struct {
	db *geoip2.Reader
}

func New(databasePath string) (*Resolver, error) {
	databasePath = strings.TrimSpace(databasePath)
	if databasePath == "" {
		return &Resolver{}, nil
	}
	db, err := geoip2.Open(databasePath)
	if err != nil {
		return nil, err
	}
	return &Resolver{db: db}, nil
}

func (r *Resolver) Lookup(ipText string) Location {
	fallback := UnknownLocation()
	if r == nil || r.db == nil {
		return fallback
	}
	ip := net.ParseIP(strings.TrimSpace(ipText))
	if ip == nil || ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() {
		return fallback
	}
	record, err := r.db.City(ip)
	if err != nil || record == nil {
		return fallback
	}
	location := Location{
		CountryCode: strings.TrimSpace(record.Country.IsoCode),
		CountryName: localizedName(record.Country.Names),
		CityName:    localizedName(record.City.Names),
	}
	if len(record.Subdivisions) > 0 {
		location.RegionName = localizedName(record.Subdivisions[0].Names)
	}
	return Normalize(location)
}

func (r *Resolver) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

func UnknownLocation() Location {
	return Location{
		CountryCode: unknown,
		CountryName: unknown,
		RegionName:  unknown,
		CityName:    unknown,
	}
}

func Normalize(location Location) Location {
	location.CountryCode = fallback(location.CountryCode)
	location.CountryName = fallback(location.CountryName)
	location.RegionName = fallback(location.RegionName)
	location.CityName = fallback(location.CityName)
	return location
}

func localizedName(names map[string]string) string {
	if names == nil {
		return ""
	}
	for _, key := range []string{"zh-CN", "zh", "en"} {
		if value := strings.TrimSpace(names[key]); value != "" {
			return value
		}
	}
	for _, value := range names {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func fallback(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return unknown
	}
	runes := []rune(value)
	if len(runes) > 128 {
		return string(runes[:128])
	}
	return value
}
