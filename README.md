ZAnnotate
=========

ZAnnotate is a Go utility that facilitates annotating large datasets
with network metadata. Right now this includes:

 * Maxmind GeoIP2
 * AS/Routing Data (based on an MRT routing table)

For example, you can add Maxmind geolocation data to a list of IPs:

	cat ips.csv | zannotate --geoip2 --geoip2-database=geoip2.mmdb


# Installation

ZAnnotate can be installed using `make install`

```shell
make install
```

or if you don't have `make` installed, you can use the following command:

```shell
cd cmd/zannotate && go install
```

Either way, this will install the `zannotate` binary in your `$GOPATH/bin` directory.

Check that it was installed correctly with:

```shell
zannotate --help
```

# Acquiring Datasets

> [!NOTE]
> URLs and instructions may change over time. These are up-to-date as of September 2025.

Below are instructions for getting datasets from the below providers.

### GeoLite2-ASN
1. [Sign-up form](https://www.maxmind.com/en/geolite2/signup) for MaxMind GeoLite Access
2. Login to your account
3. Go to the "GeoIP / GeoLite" > "Download files" section where you should see a list of available databases
4. Download the `.mmdb` files for GeoLite ASN
5. Unzip the downloaded file and test with:

```shell
echo "1.1.1.1" | zannotate --geoasn --geoasn-database=/path-to-downloaded-file/GeoLite2-ASN_20250923/GeoLite2-ASN.mmdb
```

```shell
{"ip":"1.1.1.1","geoasn":{"asn":13335,"org":"CLOUDFLARENET"}}
```
