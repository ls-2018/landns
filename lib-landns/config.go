package landns

import (
	"fmt"
	"net"
)

const (
	// DefaultTTL is the default TTL in the configuration file.
	DefaultTTL uint32 = 3600
)

// Proto is type of protocol ("tcp" or "udp").
type Proto string

// String is getter to string.
//
// String will returns "tcp" if proto is empty string.
func (p Proto) String() string {
	if string(p) == "" {
		return "tcp"
	}
	return string(p)
}

// Normalized is normalizer for Proto.
func (p Proto) Normalized() Proto {
	return Proto(p.String())
}

// Validate is validator of Proto.
//
// Returns error if value is not "tcp", "udp", or empty string.
func (p Proto) Validate() error {
	if p.String() != "" && p.String() != "tcp" && p.String() != "udp" {
		return newError(TypeArgumentError, nil, "invalid protocol: %s", p)
	}
	return nil
}

// UnmarshalText is parse text to Proto.
func (p *Proto) UnmarshalText(text []byte) error {
	if string(text) == "" {
		*p = "tcp"
	} else {
		*p = Proto(string(text))
	}

	return p.Validate()
}

// MarshalText is make bytes text.
func (p Proto) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

// SrvRecordConfig is configuration for SRV record of static zone.
type SrvRecordConfig struct {
	Service  string `yaml:"service"`
	Proto    Proto  `yaml:"proto,omitempty"`
	Priority uint16 `yaml:"priority,omitempty"`
	Weight   uint16 `yaml:"weight,omitempty"`
	Port     uint16 `yaml:"port"`
	Target   Domain `yaml:"target"`
}

// ToRecord is converter to SrvRecord.
func (s SrvRecordConfig) ToRecord(name Domain, ttl uint32) SrvRecord {
	return SrvRecord{
		Name:     Domain(fmt.Sprintf("_%s._%s.%s", s.Service, s.Proto.Normalized(), name)),
		TTL:      ttl,
		Priority: s.Priority,
		Weight:   s.Weight,
		Port:     s.Port,
		Target:   s.Target,
	}
}

// ResolverConfig is configuration for static zone.
type ResolverConfig struct {
	TTL       *uint32                      `yaml:"ttl,omitempty"`
	Addresses map[Domain][]net.IP          `yaml:"address,omitempty"`
	Cnames    map[Domain][]Domain          `yaml:"cname,omitempty"`
	Texts     map[Domain][]string          `yaml:"text,omitempty"`
	Services  map[Domain][]SrvRecordConfig `yaml:"service,omitempty"`
}
