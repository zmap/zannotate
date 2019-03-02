ZAnnotate
=========

ZAnnotate is a Go utility that facilitates annotating large datasets
with network metadata. Right now this includes:

 * Maxmind GeoIP2
 * AS/Routing Data (based on an MRT routing table)

For example, you can add Maxmind geolocation data to a list of IPs:

	cat ips.csv | zannotate --geoip2 --geoip2-database=geoip2.mmdb
