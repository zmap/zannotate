ZAnnotate
=========

ZAnnotate is a golang-based utility that facilitates annotating large datasets
with additional network metadata. Right now this includes:

 * Maxmind GeoIP2
 * AS/Routing Data (based on an MRT routing table)

For example, you can add Maxmind geolocation data to a list of IPs:

	cat ips.csv | zannotate --geoip2 --geoip-database=geoip2.mmdb
