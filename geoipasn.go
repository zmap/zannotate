package zannotate

import (
	"flag"
	"github.com/oschwald/geoip2-golang"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
)

type GeoIPASNOutput struct {
	ASN			uint 	`json:"asn,omitempty"`
	ASNOrg		string	`json:"asn_org,omitempty"`
}

type GeoIPASNAnnotatorFactory struct {
	BasePluginConf
	Path 		string
	Mode		string
}

type GeoIPASNAnnotator struct {
	Factory *GeoIPASNAnnotatorFactory
	Reader	*geoip2.Reader
	Id		int
}

func (fact *GeoIPASNAnnotatorFactory) AddFlags(flags *flag.FlagSet) {
	flags.BoolVar(&fact.Enabled, "geoasn", false, "annotate with Maxmind Geolite ASN data")
	flags.StringVar(&fact.Path, "geoasn-database", "", "path to Maxmind ASN database")
	flags.StringVar(&fact.Mode, "geoasn-mode", "mmap", "how to open database: mmap or memory")
	flags.IntVar(&fact.Threads, "geoasn-threads", 5, "how many geoASN processing threads to use")
}

func (fact *GeoIPASNAnnotatorFactory) IsEnabled() bool {
	return fact.Enabled
}

func (fact *GeoIPASNAnnotatorFactory) GetWorkers() int {
	return fact.Threads
}

func (fact *GeoIPASNAnnotatorFactory) MakeAnnotator(i int) Annotator {
	var v GeoIPASNAnnotator
	v.Factory = fact
	v.Id = i
	return &v
}

func (fact *GeoIPASNAnnotatorFactory) Initialize(conf *GlobalConf) error {
	if fact.Path == "" {
		log.Fatal("no GeoIP ASN database provided")
	}
	log.Info("Will add ASNs using ", fact.Path)
	return nil
}

func (fact *GeoIPASNAnnotatorFactory) Close() error {
	return nil
}

func (anno *GeoIPASNAnnotator) Initialize() error {
	if anno.Factory.Mode == "memory" {
		bytes, err := ioutil.ReadFile(anno.Factory.Path)
		if err != nil {
			log.Fatal("unable to open maxmind geoIP ASN database (memory): ", err)
		}
		db, err := geoip2.FromBytes(bytes)
		if err != nil {
			log.Fatal("unable to parse maxmind geoIP ASN database: ", err)
		}
		anno.Reader = db
	} else if anno.Factory.Mode == "mmap" {
		db, err := geoip2.Open(anno.Factory.Path)
		if err != nil {
			log.Fatal("unable to load maxmind geoIP ASN database: ", err)
		}
		anno.Reader = db
	} else {
		log.Fatal("invalid GeoIP ASN mode")
	}
	return nil
}

func (anno *GeoIPASNAnnotator) GetFieldName() string {
	return "geoasn"
}

func (anno *GeoIPASNAnnotator) Annotate(ip net.IP) interface{} {
	record, err := anno.Reader.ASN(ip)
	if err != nil {
		log.Error(err)
		return &GeoIPASNOutput{}
	}
	return &GeoIPASNOutput{
		ASN: record.AutonomousSystemNumber,
		ASNOrg: record.AutonomousSystemOrganization,
	}
}

func (anno *GeoIPASNAnnotator) Close() error {
	return anno.Reader.Close()
}

func init() {
	f := new (GeoIPASNAnnotatorFactory)
	RegisterAnnotator(f)
}
