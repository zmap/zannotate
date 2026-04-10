ZAnnotate
=========

ZAnnotate is a Go utility that facilitates annotating large datasets
with network metadata. Right now this includes:

 * Maxmind GeoIP2
 * AS/Routing Data (based on an MRT routing table)
 * IPInfo.io ASN and Geolocation data

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

### MaxMind GeoIP ASN and City (Formerly GeoIP2)

MaxMind provides datasets for IP geolocation and ASN data in both a free (GeoLite) and paid (GeoIP) version.
Additionally, both the GeoLite and GeoIP datasets come in two access patterns - a downloadable database file that can be queried locally and a web API.
The GeoIP module in ZAnnotate supports the local database in both GeoLite and GeoIP versions.

The following assumes you want to use the free GeoLite datasets, but the process is similar for the paid GeoIP data.

1. [Sign-up form](https://www.maxmind.com/en/geolite2/signup) for MaxMind GeoLite Access
2. Login to your account
3. Go to the "GeoIP / GeoLite" > "Download files" section and download the zip files for either GeoLite ASN or GeoLite City
   datasets.

![GeoLite Download Page](.github/readme-images/maxmind-geolite-downloads-screenshot.png)

4. Unzip, place the `.mmdb` files somewhere and test with the below.

#### MaxMind GeoIP City
```shell
echo "171.67.71.209" | ./zannotate --geoip2 --geoip2-database=./path-to-geolite2.mmdb
```

```json
{"ip":"171.67.71.209","geoip2":{"city":{"name":"Vallejo","id":5405380},"country":{"name":"United States","code":"US","id":6252001},"continent":{"name":"North America","code":"NA","id":6255149},"postal":{"code":"94590"},"latlong":{"accuracy_radius":50,"latitude":38.1043,"longitude":-122.2442,"metro_code":807,"time_zone":"America/Los_Angeles"},"represented_country":{},"registered_country":{"name":"United States","code":"US","id":6252001},"metadata":{}}}
```

#### MaxMind GeoLite ASN
```shell
 echo "171.67.71.209" | ./zannotate --geoasn --geoasn-database=/path-to-asn-file.mmdb 
```

```json
{"ip":"171.67.71.209","geoasn":{"asn":32,"org":"STANFORD"}}
```

### BGP Routing Tables
1. Go to [https://archive.routeviews.org/route-views2/bgpdata/](https://archive.routeviews.org/route-views2/bgpdata/)
2. Select a month directory (e.g. `2025.09`)
3. Select the `RIBS/` directory
4. Download a BZiped MRT file (e.g. `rib.20250923.1200.bz2`)
5. Unzip the file with:

```shell
bzip2 -d ./path-to-downloaded-file/rib.20250923.1200.bz2
```

6. Test with:

```shell
echo "1.1.1.1" | ./zannotate --routing --routing-mrt-file=/tmp/rib.20250923.1200
```

```json
{"ip":"1.1.1.1","routing":{"prefix":"1.1.1.0/24","asn":13335,"path":[3561,209,3356,13335]}}
```
### IPInfo.io
IPInfo.io provides a free dataset that includes ASN and geolocation data, scoped to the country level. Paid tiers provide
more granular geo-location data.

1. Sign up for a free account at [IPInfo.io](https://ipinfo.io/signup).
2. Navigate to the [Data Download page](https://ipinfo.io/dashboard/downloads)
3. Download the `mmdb` file
   ![IPInfo Download Page](.github/readme-images/ipinfoio-data-downloads-screenshot.png)
4. Example CLI usage

```shell
echo "1.1.1.1" | ./zannotate --ipinfo --ipinfo-database=./path-to-ipinfo-db.mmdb
```

```json
{"ip":"1.1.1.1","ipinfo":{"country":"Australia","country_code":"AU","continent":"Oceania","continent_code":"OC","asn":"AS13335","as_name":"Cloudflare, Inc.","as_domain":"cloudflare.com"}}
```

### Spur IP Enrichment + Intelligence

[spur.us](https://spur.us/) provides per‑IP intelligence such as ASN and organization, infrastructure classification (e.g., datacenter, CDN, mobile), and geolocation metadata. 
We can query [spur.us](https://spur.us/) alongside other sources to enrich annotations and help identify datacenter/Anycast deployments, CDNs, and ISP ownership.

0. You'll need an API key from Spur to enable ZAnnotate to enrich IPs with their dataset as we'll need to make an API request for each IP address to be enriched.
Depending on current pricing, you may need to sign up for a paid account.
Check out [spur.us/pricing](https://spur.us/pricing) for more details.
1. Once you have an API key, set it as an environment variable in your current shell session:

```shell
export SPUR_API_KEY=your_api_key_here
```
(If you want to make this permanent, add the above line to your shell profile, e.g. `~/.bashrc` or `~/.zshrc`)
2. Test with:

```shell
$ echo "1.1.1.1" | ./zannotate --spur
```

Example Output:
```shell
{"ip":"1.1.1.1","spur":{"as":{"number":13335,"organization":"Cloudflare, Inc."},"infrastructure":"DATACENTER","ip":"1.1.1.1","location":{"city":"Anycast","country":"ZZ","state":"Anycast"},"organization":"Taguchi Digital Marketing System"}}
```

### Greynoise Psychic
[Greynoise](https://greynoise.io) is an IP intelligence feed that provides metadata like threat classification and associated CVE's.
Their Psychic data downloads provide their data feed in a database suitable for offline data enrichment.
To use their download with `zannotate`, you'll want to download an `.mmdb` formatted file using your GreyNoise API key. 
As of April 2026, signing up with a free account gives access to data downloads.

0. Sign up for a free GreyNoise account [here](https://www.greynoise.io).
1. Copy API key from the appropriate [section of your account](https://viz.greynoise.io/workspace/api-key).
2. Download a `mmdb` file. Details on download parameters 
(The below command is for downloading data for a single date - April 7th, 2026 - you can also download data for a range of days and for models of various levels of detail.
See GreyNoise's Psychic [documentation](https://psychic.labs.greynoise.io) for more details.
```shell
curl -H "key: GREYNOISE_API_KEY_HERE" \
           https://psychic.labs.greynoise.io/v1/psychic/download/2026-04-07/3/mmdb \
           -o /tmp/m3.mmdb
```

3. Test GreyNoise data enrichment:

> [!NOTE]
> The below examples are using the exact data download from the above `curl` command. What results you see will depend on the data downloaded.

```shell
echo "14.1.105.157" | zannotate --greynoise --greynoise-database=/tmp/m3.mmdb  
````
Example Output:
```json
{"greynoise":{"classification":"malicious","cves":["CVE-2015-2051","CVE-2016-20016","CVE-2018-10561","CVE-2018-10562","CVE-2016-6277","CVE-2024-12847"],"date":"2026-04-07","handshake_complete":true,"last_seen":"2026-04-07T00:00:00Z","seen":true,"tags":["Mirai TCP Scanner","Mirai","Telnet Protocol","Generic IoT Default Password Attempt","Web Crawler","Generic Suspicious Linux Command in Request","HNAP Crawler","Telnet Login Attempt","D-Link Devices HNAP SOAPAction Header RCE Attempt","MVPower CCTV DVR RCE CVE-2016-20016 Attempt","JAWS Webserver RCE","GPON CVE-2018-10561 Router Worm","Generic ${IFS} Use in RCE Attempt","CCTV-DVR RCE","NETGEAR Command Injection CVE-2016-6277","NETGEAR DGN setup.cgi CVE-2024-12847 Command Execution Attempt","CGI Script Scanner"],"actor":"unknown"},"ip":"14.1.105.157"}
```

Note that many IPs will not be in the GreyNoise dataset, so you may see output like the following:
```shell
echo "1.1.1.1" | zannotate --greynoise --greynoise-database=/tmp/m3.mmdb  
```

```json
{"ip":"1.1.1.1","greynoise":null}
```


# Input/Output

## Output
By default, ZAnnotate reads new-line delimited IP addresses from standard input and outputs a JSON object per line to standard output like:

```shell
echo "1.1.1.1" | zannotate --rdns --geoasn --geoasn-database=/path-to-geo-asn.mmdb
```

```json
{"ip":"1.1.1.1","geoasn":{"asn":13335,"org":"CLOUDFLARENET"},"rdns":{"domain_names":["one.one.one.one"]}}
```

If an IP address cannot be annotated, either because of an error or lack of data, there will be an empty field for that annotation.
For example, if an IP address is private and therefore has no RDNS or ASN data, the output will look like:
```shell
echo "127.0.0.1" | zannotate --rdns --geoasn --geoasn-database=/path-to-geo-asn.mmdb
```

```json
{"geoasn":{},"rdns":{},"ip":"127.0.0.1"}
```

### Censys
Censys provides internet-wide host and network data, including information on what services are running on an IP, what TLS certificates it has, and more.
They offer a free tier that allows for a limited number of queries per month, which can be used to enrich IP annotations with Censys data.

Note: The free account (as of April 2026) allows a single concurrent request and 100 requests per month.
Once you've used all your credits, the `censys:` annotation will be empty for any following IPs until your credits refill the following month.

0. Create an account at [Censys.io](https://censys.io) and get a Personal Access Token (PAT) from the Personal Settings > Personal Access Tokens section.
> [!NOTE]
> Censys's free tier allows for a single concurrent request.
> Therefore you wouldn't want to modify the `--censys-threads` flag unless you pay for a higher tier and can then set it accordingly.
1. Annotate with Censys data:

```shell
echo "8.8.8.8" | zannotate --censys --censys-pat="CENSYS_PAT_HERE"   
```

```json
{"ip":"8.8.8.8","censys":{"result":{"resource":{"whois":{"network":{"name":"Google LLC","cidrs":["8.8.8.0/24"],"created":"2023-12-28T00:00:00Z","updated":"2023-12-28T00:00:00Z","allocation_type":"ALLOCATION","handle":"GOGL"},"organization":{"state":"CA","postal_code":"94043","country":"US","tech_contacts":[{"handle":"ZG39-ARIN","name":"Google LLC","email":"arin-contact@google.com"}],"handle":"GOGL","street":"1600 Amphitheatre Parkway","abuse_contacts":[{"handle":"ABUSE5250-ARIN","name":"Abuse","email":"network-abuse@google.com"}],"admin_contacts":[{"handle":"ZG39-ARIN","name":"Google LLC","email":"arin-contact@google.com"}],"name":"Google LLC","city":"Mountain View"}},"services":[{"port":53,"protocol":"DNS","transport_protocol":"udp","ip":"8.8.8.8","scan_time":"2026-04-10T02:55:53Z","dns":{"server_type":"forwarding","resolves_correctly":true,"answers":[{"name":"ip.parrotdns.com.","response":"35.202.119.40","type":"a"},{"name":"ip.parrotdns.com.","response":"74.125.183.150","type":"a"}],"questions":[{"name":"ip.parrotdns.com.","response":";ip.parrotdns.com.\tIN\t A","type":"a"}],"edns":{"do":true,"udp":512},"r_code":"success"}},{"transport_protocol":"quic","ip":"8.8.8.8","scan_time":"2026-04-09T23:36:58Z","port":443,"protocol":"UNKNOWN"},{"scan_time":"2026-04-10T03:23:08Z","tls":{"fingerprint_sha256":"a766f314fb869eae5d9ac17603c02aba415dc80f00a7dbbccf7ec0d3ad18142c","presented_chain":[{"fingerprint_sha256":"e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","subject_dn":"C=US, O=Google Trust Services, CN=WR2","issuer_dn":"C=US, O=Google Trust Services LLC, CN=GTS Root R1"},{"fingerprint_sha256":"3ee0278df71fa3c125c4cd487f01d774694e6fc57e0cd94c24efd769133918e5","subject_dn":"C=US, O=Google Trust Services LLC, CN=GTS Root R1","issuer_dn":"C=BE, O=GlobalSign nv-sa, OU=Root CA, CN=GlobalSign Root CA"}]},"endpoints":[{"hostname":"8.8.8.8","port":443,"path":"/","endpoint_type":"HTTP","transport_protocol":"tcp","scan_time":"2026-04-10T03:23:08Z","http":{"headers":{"Access-Control-Allow-Origin":{"headers":["*"]},"Content-Length":{"headers":["216"]},"Content-Type":{"headers":["text/html; charset=UTF-8"]},"Location":{"headers":["https://dns.google/"]},"X-XSS-Protection":{"headers":["0"]},"Alt-Svc":{"headers":["h3=\":443\"; ma=2592000,h3-29=\":443\"; ma=2592000"]},"Date":{"headers":["\u003cREDACTED\u003e"]},"Server":{"headers":["HTTP server (unknown)"]},"X-Content-Type-Options":{"headers":["nosniff"]},"X-Frame-Options":{"headers":["SAMEORIGIN"]}},"html_tags":["\u003cTITLE\u003e302 Moved\u003c/TITLE\u003e","\u003cmeta http-equiv=\"content-type\" content=\"text/html;charset=utf-8\"\u003e"],"html_title":"302 Moved","supported_versions":["HTTP/1.1","HTTP/2"],"redirect_chain":[{"reason":"HTTP_3XX","scheme":"https","hostname":"dns.google","port":443,"transport_protocol":"tcp","path":"/"}],"protocol":"HTTP/1.1","body_size":216,"body":"\u003cHTML\u003e\u003cHEAD\u003e\u003cmeta http-equiv=\"content-type\" content=\"text/html;charset=utf-8\"\u003e\n\u003cTITLE\u003e302 Moved\u003c/TITLE\u003e\u003c/HEAD\u003e\u003cBODY\u003e\n\u003cH1\u003e302 Moved\u003c/H1\u003e\nThe document has moved\n\u003cA HREF=\"https://dns.google/\"\u003ehere\u003c/A\u003e.\r\n\u003c/BODY\u003e\u003c/HTML\u003e\r\n","body_hash_sha256":"4d2df97f002e1d1905cf64366eddaa9a57f2f99b3cc7101c3bebff18428ed894","body_hash_sha1":"1fd84b37b709256752fe1f865f86b5bec05cf712","uri":"https://8.8.8.8/","status_code":302,"status_reason":"Found"},"ip":"8.8.8.8"}],"port":443,"protocol":"HTTP","transport_protocol":"tcp","cert":{"spki_fingerprint_sha256":"8b7b8516c3e2fc92c6f3e5bc09acfa2f7167da932f56feccebb46e3423819335","parsed":{"version":3,"serial_number":"196021321340889868047382337029571940953","issuer_dn":"C=US, O=Google Trust Services, CN=WR2","subject_dn":"CN=dns.google","subject":{"common_name":["dns.google"]},"signature":{"signature_algorithm":{"name":"SHA256-RSA","oid":"1.2.840.113549.1.1.11"},"value":"1702fbe39032cfa6873e5407a23930eb393e4694595867b702002fa23f765eb3f916929c48e45dfbedaf4587b27eb519cffd4ae038ba8fd48c0cd39cad72ba1b3c8c374ccc4a90a270af0d2acc22196bf53f90fc9b677e461dee581cb2cef2ea74c83373067b4c5e8cd657097862af290c1bfd06e59186962f62720c3ba148ed6d13b8b3f67799f8804ebe165b057b1ed0f1468f7def4a71fb64a7b0ea4c2f2e36d72721978086dbbcbb5c68cd90719d1637cd8fbdb2d4802f6a92e75dfd838239df523e70c691a7036d8bfdd38147cbe51e309edff1624da9e6f7297db6b44aa461b7ee906e7b76718e6471c685ba2aa5bc5f5cded8800075f48147868e5876","valid":true},"serial_number_hex":"9378554fedf3284109d19c47baf91259","issuer":{"common_name":["WR2"],"country":["US"],"organization":["Google Trust Services"]},"subject_key_info":{"key_algorithm":{"oid":"1.2.840.113549.1.1.1","name":"RSA"},"rsa":{"length":2048,"exponent":65537,"modulus":"cd8b665690aeafaf746bd79ac03ea211b56ea6abc6225934cf52060b274fc240288d408803e1c657fb607e44f8f099b149aee462ea27f8373d0bf42d3a4e95f54a873be56f51ceea712331b3b08e43c98abb20bb77216a9afac6ff5dc05ca1ad636689fd4c7e1c654bf7fcf36e26a6317686b8f6a0ee8392a5e5c5118a9d4a84a1767425f9f75574676d17f589d6b070abd90a794f407e44f6b1e13d38d66029129f40c3adfdedddd375b7d5717b5de77412262bfecbb26946193d2054eb216baab47536f711f15e3187668f2ec8ee01ee7e4c8061a6d2996d2e1d8ca5b32db97c0ec9d89834aac64e5c317e37f8df4274114c09a8b777be36d4b66c628987c9"},"fingerprint_sha256":"30e96aa0b7e1278efc3755d03ad3e9284ae673ed94fac81847140682b6a868bd"},"validity_period":{"not_before":"2026-03-16T08:39:59Z","not_after":"2026-06-08T08:39:58Z","length_seconds":7257600}},"validation_level":"dv","parse_status":"success","fingerprint_md5":"c6bf716b5eadfa866d9799242e5985b9","ct":{"entries":{"trustasia_log_2026_b":{"added_to_ct_at":"2026-03-31T15:14:43Z","ct_to_censys_at":"2026-03-31T15:25:41Z","index":1207176426},"google_argon_2026_h1":{"ct_to_censys_at":"2026-03-31T15:19:30Z","index":2742627595,"added_to_ct_at":"2026-03-31T15:14:42Z"},"letsencrypt_ct_willow_2026_h1":{"index":909009584,"added_to_ct_at":"2026-03-31T16:16:20Z","ct_to_censys_at":"2026-03-31T16:25:33Z"},"sectigo_elephant_2026_h1":{"index":1994983495,"added_to_ct_at":"2026-03-31T15:14:42Z","ct_to_censys_at":"2026-03-31T15:18:21Z"},"trustasia_log_2026_a":{"index":1186766901,"added_to_ct_at":"2026-03-31T15:14:45Z","ct_to_censys_at":"2026-03-31T15:25:49Z"},"cloudflare_nimbus_2026":{"index":3554816787,"added_to_ct_at":"2026-04-09T09:26:12Z","ct_to_censys_at":"2026-04-09T10:16:18Z"},"google_xenon_2026_h1":{"index":2479660224,"added_to_ct_at":"2026-03-31T15:14:42Z","ct_to_censys_at":"2026-03-31T15:25:49Z"},"letsencrypt_ct_sycamore_2026_h1":{"added_to_ct_at":"2026-03-31T16:16:12Z","ct_to_censys_at":"2026-03-31T16:26:41Z","index":906566715},"sectigo_tiger_2026_h1":{"index":1934427065,"added_to_ct_at":"2026-03-31T15:14:42Z","ct_to_censys_at":"2026-03-31T15:23:32Z"}}},"parent_spki_subject_fingerprint_sha256":"95b148afc4c249d314067527813d43973574f8e11a905040c881510026ae74f9","fingerprint_sha1":"a98e2d29813e7c927338d3e7b2a454fa498516ee","parent_spki_fingerprint_sha256":"95b148afc4c249d314067527813d43973574f8e11a905040c881510026ae74f9","modified_at":"2026-04-09T10:16:19Z","spki_subject_fingerprint_sha256":"8b7b8516c3e2fc92c6f3e5bc09acfa2f7167da932f56feccebb46e3423819335","fingerprint_sha256":"a766f314fb869eae5d9ac17603c02aba415dc80f00a7dbbccf7ec0d3ad18142c","tbs_no_ct_fingerprint_sha256":"3f4f4d4b4a89bd77b293ec8e4f65144d949cc6a581eb0902e298154cfb439853","names":["*.dns.google.com","8.8.4.4","8.8.8.53","8.8.8.8","8888.google","dns.google","dns.google.com","dns64.dns.google"],"validation":{"nss":{"chains":[{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","d947432abde7b7fa90fc2e6b59101b1280e0e1c7e4e40fa3c6887fff57a7f4cf"]},{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","3ee0278df71fa3c125c4cd487f01d774694e6fc57e0cd94c24efd769133918e5","ebd41040e4bb3ec742c9e381d31ef2a41a48b6685c96e7cef3c1df6cd4331c99"]}],"parents":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4"],"type":"leaf","is_valid":true,"ever_valid":true,"has_trusted_path":true,"had_trusted_path":true},"microsoft":{"parents":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4"],"type":"leaf","is_valid":true,"ever_valid":true,"has_trusted_path":true,"had_trusted_path":true,"chains":[{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","2a575471e31340bc21581cbd2cf13e158463203ece94bcf9d3cc196bf09a5472"]},{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","d947432abde7b7fa90fc2e6b59101b1280e0e1c7e4e40fa3c6887fff57a7f4cf"]},{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","3ee0278df71fa3c125c4cd487f01d774694e6fc57e0cd94c24efd769133918e5","ebd41040e4bb3ec742c9e381d31ef2a41a48b6685c96e7cef3c1df6cd4331c99"]}]},"apple":{"type":"leaf","is_valid":true,"ever_valid":true,"has_trusted_path":true,"had_trusted_path":true,"chains":[{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","d947432abde7b7fa90fc2e6b59101b1280e0e1c7e4e40fa3c6887fff57a7f4cf"]},{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","3ee0278df71fa3c125c4cd487f01d774694e6fc57e0cd94c24efd769133918e5","ebd41040e4bb3ec742c9e381d31ef2a41a48b6685c96e7cef3c1df6cd4331c99"]}],"parents":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4"]},"chrome":{"has_trusted_path":true,"had_trusted_path":true,"chains":[{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","d947432abde7b7fa90fc2e6b59101b1280e0e1c7e4e40fa3c6887fff57a7f4cf"]}],"parents":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4"],"type":"leaf","is_valid":true,"ever_valid":true}},"revocation":{"ocsp":{"reason":"unspecified"},"crl":{"reason":"unspecified"}},"ever_seen_in_scan":true,"added_at":"2026-03-28T06:33:10Z","validated_at":"2026-04-09T17:11:38Z","tbs_fingerprint_sha256":"c1208c818437581351fd9384eaaf5ec0ff61b5c1b436a5e52a4c2f74e947dfa3"},"ip":"8.8.8.8"},{"scan_time":"2026-04-09T20:20:13Z","tls":{"fingerprint_sha256":"a766f314fb869eae5d9ac17603c02aba415dc80f00a7dbbccf7ec0d3ad18142c","presented_chain":[{"fingerprint_sha256":"e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","subject_dn":"C=US, O=Google Trust Services, CN=WR2","issuer_dn":"C=US, O=Google Trust Services LLC, CN=GTS Root R1"},{"fingerprint_sha256":"3ee0278df71fa3c125c4cd487f01d774694e6fc57e0cd94c24efd769133918e5","subject_dn":"C=US, O=Google Trust Services LLC, CN=GTS Root R1","issuer_dn":"C=BE, O=GlobalSign nv-sa, OU=Root CA, CN=GlobalSign Root CA"}]},"port":853,"protocol":"UNKNOWN","transport_protocol":"tcp","cert":{"fingerprint_sha256":"a766f314fb869eae5d9ac17603c02aba415dc80f00a7dbbccf7ec0d3ad18142c","tbs_no_ct_fingerprint_sha256":"3f4f4d4b4a89bd77b293ec8e4f65144d949cc6a581eb0902e298154cfb439853","names":["*.dns.google.com","8.8.4.4","8.8.8.53","8.8.8.8","8888.google","dns.google","dns.google.com","dns64.dns.google"],"validation_level":"dv","parse_status":"success","fingerprint_sha1":"a98e2d29813e7c927338d3e7b2a454fa498516ee","tbs_fingerprint_sha256":"c1208c818437581351fd9384eaaf5ec0ff61b5c1b436a5e52a4c2f74e947dfa3","spki_fingerprint_sha256":"8b7b8516c3e2fc92c6f3e5bc09acfa2f7167da932f56feccebb46e3423819335","parent_spki_fingerprint_sha256":"95b148afc4c249d314067527813d43973574f8e11a905040c881510026ae74f9","validation":{"nss":{"parents":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4"],"type":"leaf","is_valid":true,"ever_valid":true,"has_trusted_path":true,"had_trusted_path":true,"chains":[{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","d947432abde7b7fa90fc2e6b59101b1280e0e1c7e4e40fa3c6887fff57a7f4cf"]},{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","3ee0278df71fa3c125c4cd487f01d774694e6fc57e0cd94c24efd769133918e5","ebd41040e4bb3ec742c9e381d31ef2a41a48b6685c96e7cef3c1df6cd4331c99"]}]},"microsoft":{"had_trusted_path":true,"chains":[{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","2a575471e31340bc21581cbd2cf13e158463203ece94bcf9d3cc196bf09a5472"]},{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","d947432abde7b7fa90fc2e6b59101b1280e0e1c7e4e40fa3c6887fff57a7f4cf"]},{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","3ee0278df71fa3c125c4cd487f01d774694e6fc57e0cd94c24efd769133918e5","ebd41040e4bb3ec742c9e381d31ef2a41a48b6685c96e7cef3c1df6cd4331c99"]}],"parents":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4"],"type":"leaf","is_valid":true,"ever_valid":true,"has_trusted_path":true},"apple":{"has_trusted_path":true,"had_trusted_path":true,"chains":[{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","d947432abde7b7fa90fc2e6b59101b1280e0e1c7e4e40fa3c6887fff57a7f4cf"]},{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","3ee0278df71fa3c125c4cd487f01d774694e6fc57e0cd94c24efd769133918e5","ebd41040e4bb3ec742c9e381d31ef2a41a48b6685c96e7cef3c1df6cd4331c99"]}],"parents":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4"],"type":"leaf","is_valid":true,"ever_valid":true},"chrome":{"has_trusted_path":true,"had_trusted_path":true,"chains":[{"sha256fp":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4","d947432abde7b7fa90fc2e6b59101b1280e0e1c7e4e40fa3c6887fff57a7f4cf"]}],"parents":["e6fe22bf45e4f0d3b85c59e02c0f495418e1eb8d3210f788d48cd5e1cb547cd4"],"type":"leaf","is_valid":true,"ever_valid":true}},"revocation":{"ocsp":{"reason":"unspecified"},"crl":{"reason":"unspecified"}},"added_at":"2026-03-28T06:33:10Z","fingerprint_md5":"c6bf716b5eadfa866d9799242e5985b9","parsed":{"serial_number":"196021321340889868047382337029571940953","issuer":{"common_name":["WR2"],"country":["US"],"organization":["Google Trust Services"]},"subject_key_info":{"fingerprint_sha256":"30e96aa0b7e1278efc3755d03ad3e9284ae673ed94fac81847140682b6a868bd","key_algorithm":{"name":"RSA","oid":"1.2.840.113549.1.1.1"},"rsa":{"exponent":65537,"modulus":"cd8b665690aeafaf746bd79ac03ea211b56ea6abc6225934cf52060b274fc240288d408803e1c657fb607e44f8f099b149aee462ea27f8373d0bf42d3a4e95f54a873be56f51ceea712331b3b08e43c98abb20bb77216a9afac6ff5dc05ca1ad636689fd4c7e1c654bf7fcf36e26a6317686b8f6a0ee8392a5e5c5118a9d4a84a1767425f9f75574676d17f589d6b070abd90a794f407e44f6b1e13d38d66029129f40c3adfdedddd375b7d5717b5de77412262bfecbb26946193d2054eb216baab47536f711f15e3187668f2ec8ee01ee7e4c8061a6d2996d2e1d8ca5b32db97c0ec9d89834aac64e5c317e37f8df4274114c09a8b777be36d4b66c628987c9","length":2048}},"signature":{"signature_algorithm":{"name":"SHA256-RSA","oid":"1.2.840.113549.1.1.11"},"value":"1702fbe39032cfa6873e5407a23930eb393e4694595867b702002fa23f765eb3f916929c48e45dfbedaf4587b27eb519cffd4ae038ba8fd48c0cd39cad72ba1b3c8c374ccc4a90a270af0d2acc22196bf53f90fc9b677e461dee581cb2cef2ea74c83373067b4c5e8cd657097862af290c1bfd06e59186962f62720c3ba148ed6d13b8b3f67799f8804ebe165b057b1ed0f1468f7def4a71fb64a7b0ea4c2f2e36d72721978086dbbcbb5c68cd90719d1637cd8fbdb2d4802f6a92e75dfd838239df523e70c691a7036d8bfdd38147cbe51e309edff1624da9e6f7297db6b44aa461b7ee906e7b76718e6471c685ba2aa5bc5f5cded8800075f48147868e5876","valid":true},"serial_number_hex":"9378554fedf3284109d19c47baf91259","version":3,"issuer_dn":"C=US, O=Google Trust Services, CN=WR2","subject_dn":"CN=dns.google","subject":{"common_name":["dns.google"]},"validity_period":{"not_after":"2026-06-08T08:39:58Z","length_seconds":7257600,"not_before":"2026-03-16T08:39:59Z"}},"ct":{"entries":{"letsencrypt_ct_sycamore_2026_h1":{"index":906566715,"added_to_ct_at":"2026-03-31T16:16:12Z","ct_to_censys_at":"2026-03-31T16:26:41Z"},"sectigo_tiger_2026_h1":{"added_to_ct_at":"2026-03-31T15:14:42Z","ct_to_censys_at":"2026-03-31T15:23:32Z","index":1934427065},"trustasia_log_2026_a":{"index":1186766901,"added_to_ct_at":"2026-03-31T15:14:45Z","ct_to_censys_at":"2026-03-31T15:25:49Z"},"trustasia_log_2026_b":{"index":1207176426,"added_to_ct_at":"2026-03-31T15:14:43Z","ct_to_censys_at":"2026-03-31T15:25:41Z"},"letsencrypt_ct_willow_2026_h1":{"index":909009584,"added_to_ct_at":"2026-03-31T16:16:20Z","ct_to_censys_at":"2026-03-31T16:25:33Z"},"sectigo_elephant_2026_h1":{"ct_to_censys_at":"2026-03-31T15:18:21Z","index":1994983495,"added_to_ct_at":"2026-03-31T15:14:42Z"},"cloudflare_nimbus_2026":{"index":3554816787,"added_to_ct_at":"2026-04-09T09:26:12Z","ct_to_censys_at":"2026-04-09T10:16:18Z"},"google_argon_2026_h1":{"added_to_ct_at":"2026-03-31T15:14:42Z","ct_to_censys_at":"2026-03-31T15:19:30Z","index":2742627595},"google_xenon_2026_h1":{"index":2479660224,"added_to_ct_at":"2026-03-31T15:14:42Z","ct_to_censys_at":"2026-03-31T15:25:49Z"}}},"ever_seen_in_scan":true,"validated_at":"2026-04-09T17:11:38Z","spki_subject_fingerprint_sha256":"8b7b8516c3e2fc92c6f3e5bc09acfa2f7167da932f56feccebb46e3423819335","modified_at":"2026-04-09T10:16:19Z","parent_spki_subject_fingerprint_sha256":"95b148afc4c249d314067527813d43973574f8e11a905040c881510026ae74f9"},"ip":"8.8.8.8"}],"service_count":4,"dns":{"reverse_dns":{"resolve_time":"2026-04-09T23:27:20Z"},"names":["zzz.xn--9kqt69cipkc3t.xn--fiqs8s","xn--jlqt95e.xn--xhqp98c.xn--9kqt69cipkc3t.xn--fiqs8s","xn--jlqt95e.mmm.xn--9kqt69cipkc3t.xn--fiqs8s","aaa.bbb.xn--9kqt69cipkc3t.xn--fiqs8s","aaa.xn--9kqt69cipkc3t.xn--fiqs8s","xn--6qq79v.xxx.xn--9kqt69cipkc3t.xn--fiqs8s","xn--tiq9z.xn--9kqt69cipkc3t.xn--fiqs8s","abc.xn--9kqt69cipkc3t.xn--fiqs8s","culturama.fun","kkpsolutions.ca","mundoethereum.ar","eyihz.com.cfd","fcu8.com.cfd","fcu.re.mw","3000.gleeze.com","yang.com.cfd","ewfrfwe.xyz","nanatest.xyz","106534.fangji123.xyz","prod.bumastemra.salt.ws","oldcraft.top","mycatsforever.top","www.wenshan.eu.org","www.yangergou.eu.org","kxsw1.yangxiaoyong.eu.org","test.lblb.eu.org","bbs.lblb.eu.org","chiyi4488.eu.org","dev-medicapt.org","status.moldowa.online","portal.moldowa.online","geekflare.org","mail.moldowa.online","3g.moldowa.online","masterscursos.online","www.marcoshandgemaakteschoenen.nl","dinalconcept.com.ng","awsentry.aws.corpinter.net","sdjc.oesoc.net","nxa.de5.net","exx.ddns.net","demech.net","qqqq.ddns-ip.net","woxo.de5.net","firewall.de5.net","bd5ww.de5.net","gfs.asdfhjkfg.casacam.net","chat05.asdfhjkfg.casacam.net","wmx.de5.net","2025tg.de5.net","ipsj.dy02.ddns-ip.net","jklqwe.de5.net","xykkk.de5.net","demo-lab.net","zkfnas.de5.net","dingyue.kingyu520.ddns-ip.net","pcdn.ddns-ip.net","hanyc.ddns-ip.net","dns.abilix.ddns-ip.net","awin.ddns-ip.net","llll.ddns-ip.net","sslvpn.ddns-ip.net","www.cotown.net","xsc.de5.net","cliunou.yiyuyazhu.de5.net","yiyuyazhu.de5.net","erpqt9.coscous.net","zana.ma","cashwins.ma","bistro.ma","naciha.ma","mgouna.ma","www.luxmarket.ma","lamis.ma","www.manara2.ma","glitch.ma","geocapital.ma","www.honestconcepts.ma","fle.ma","endm.ma","www.vonapartis.gr","local.snapshot.jrkyushu.co.jp","dns.diamondking.in","test.evalvsn.dedyn.io","thinkspace.thinkloud.de","tmatena.de","tknjsfd.gq","youryogaagent.com","rhddyndns.com","turabet727.com","qbets.uk.com","www.supfeel.com","gossettmktg.com","gosolaroh.com","sheerr-market.com","gortorg.com","mybabysamples.com","d.xj.mydeertrip.com","product.mydeertrip.com","lygjweb.com"],"forward_dns":{"sslvpn.ddns-ip.net":{"resolve_time":"2026-04-09T14:45:12Z","name":"sslvpn.ddns-ip.net","record_type":"a"},"tmatena.de":{"resolve_time":"2026-04-09T14:41:06Z","name":"tmatena.de","record_type":"a"},"www.supfeel.com":{"name":"www.supfeel.com","record_type":"a","resolve_time":"2026-04-09T14:32:18Z"},"xsc.de5.net":{"resolve_time":"2026-04-09T14:45:10Z","name":"xsc.de5.net","record_type":"a"},"xykkk.de5.net":{"resolve_time":"2026-04-09T14:45:20Z","name":"xykkk.de5.net","record_type":"a"},"yiyuyazhu.de5.net":{"name":"yiyuyazhu.de5.net","record_type":"a","resolve_time":"2026-04-09T14:45:10Z"},"dns.abilix.ddns-ip.net":{"resolve_time":"2026-04-09T14:45:13Z","name":"dns.abilix.ddns-ip.net","record_type":"a"},"ipsj.dy02.ddns-ip.net":{"name":"ipsj.dy02.ddns-ip.net","record_type":"a","resolve_time":"2026-04-09T14:45:23Z"},"bd5ww.de5.net":{"name":"bd5ww.de5.net","record_type":"a","resolve_time":"2026-04-09T14:45:36Z"},"geekflare.org":{"resolve_time":"2026-04-09T14:51:27Z","name":"geekflare.org","record_type":"a"},"gfs.asdfhjkfg.casacam.net":{"resolve_time":"2026-04-09T14:45:36Z","name":"gfs.asdfhjkfg.casacam.net","record_type":"a"},"llll.ddns-ip.net":{"name":"llll.ddns-ip.net","record_type":"a","resolve_time":"2026-04-09T14:45:12Z"},"mail.moldowa.online":{"resolve_time":"2026-04-09T14:51:18Z","name":"mail.moldowa.online","record_type":"a"},"status.moldowa.online":{"resolve_time":"2026-04-09T14:51:28Z","name":"status.moldowa.online","record_type":"a"},"test.evalvsn.dedyn.io":{"name":"test.evalvsn.dedyn.io","record_type":"a","resolve_time":"2026-04-09T14:42:16Z"},"www.honestconcepts.ma":{"record_type":"a","resolve_time":"2026-04-09T14:43:53Z","name":"www.honestconcepts.ma"},"106534.fangji123.xyz":{"resolve_time":"2026-04-09T14:56:09Z","name":"106534.fangji123.xyz","record_type":"a"},"awsentry.aws.corpinter.net":{"resolve_time":"2026-04-09T14:49:05Z","name":"awsentry.aws.corpinter.net","record_type":"a"},"bistro.ma":{"resolve_time":"2026-04-09T14:44:00Z","name":"bistro.ma","record_type":"a"},"demo-lab.net":{"resolve_time":"2026-04-09T14:45:19Z","name":"demo-lab.net","record_type":"a"},"naciha.ma":{"record_type":"a","resolve_time":"2026-04-09T14:43:57Z","name":"naciha.ma"},"www.manara2.ma":{"resolve_time":"2026-04-09T14:43:54Z","name":"www.manara2.ma","record_type":"a"},"xn--6qq79v.xxx.xn--9kqt69cipkc3t.xn--fiqs8s":{"name":"xn--6qq79v.xxx.xn--9kqt69cipkc3t.xn--fiqs8s","record_type":"a","resolve_time":"2026-04-10T03:21:39Z"},"xn--jlqt95e.mmm.xn--9kqt69cipkc3t.xn--fiqs8s":{"name":"xn--jlqt95e.mmm.xn--9kqt69cipkc3t.xn--fiqs8s","record_type":"a","resolve_time":"2026-04-10T03:42:47Z"},"2025tg.de5.net":{"resolve_time":"2026-04-09T14:45:24Z","name":"2025tg.de5.net","record_type":"a"},"aaa.xn--9kqt69cipkc3t.xn--fiqs8s":{"resolve_time":"2026-04-10T03:33:52Z","name":"aaa.xn--9kqt69cipkc3t.xn--fiqs8s","record_type":"a"},"abc.xn--9kqt69cipkc3t.xn--fiqs8s":{"resolve_time":"2026-04-10T02:06:51Z","name":"abc.xn--9kqt69cipkc3t.xn--fiqs8s","record_type":"a"},"product.mydeertrip.com":{"resolve_time":"2026-04-09T14:28:36Z","name":"product.mydeertrip.com","record_type":"a"},"qbets.uk.com":{"resolve_time":"2026-04-09T14:33:21Z","name":"qbets.uk.com","record_type":"a"},"portal.moldowa.online":{"resolve_time":"2026-04-09T14:51:28Z","name":"portal.moldowa.online","record_type":"a"},"xn--jlqt95e.xn--xhqp98c.xn--9kqt69cipkc3t.xn--fiqs8s":{"resolve_time":"2026-04-10T03:45:14Z","name":"xn--jlqt95e.xn--xhqp98c.xn--9kqt69cipkc3t.xn--fiqs8s","record_type":"a"},"youryogaagent.com":{"resolve_time":"2026-04-09T14:35:06Z","name":"youryogaagent.com","record_type":"a"},"zana.ma":{"resolve_time":"2026-04-09T14:44:04Z","name":"zana.ma","record_type":"a"},"3g.moldowa.online":{"resolve_time":"2026-04-09T14:51:18Z","name":"3g.moldowa.online","record_type":"a"},"cashwins.ma":{"resolve_time":"2026-04-09T14:44:03Z","name":"cashwins.ma","record_type":"a"},"erpqt9.coscous.net":{"resolve_time":"2026-04-09T14:45:01Z","name":"erpqt9.coscous.net","record_type":"a"},"gosolaroh.com":{"resolve_time":"2026-04-09T14:31:54Z","name":"gosolaroh.com","record_type":"a"},"kkpsolutions.ca":{"resolve_time":"2026-04-09T23:02:13Z","name":"kkpsolutions.ca","record_type":"a"},"mgouna.ma":{"resolve_time":"2026-04-09T14:43:56Z","name":"mgouna.ma","record_type":"a"},"www.wenshan.eu.org":{"resolve_time":"2026-04-09T14:52:20Z","name":"www.wenshan.eu.org","record_type":"a"},"lygjweb.com":{"resolve_time":"2026-04-09T14:27:27Z","name":"lygjweb.com","record_type":"a"},"mundoethereum.ar":{"resolve_time":"2026-04-09T17:18:25Z","name":"mundoethereum.ar","record_type":"a"},"mycatsforever.top":{"resolve_time":"2026-04-09T14:54:45Z","name":"mycatsforever.top","record_type":"a"},"ewfrfwe.xyz":{"resolve_time":"2026-04-09T14:56:36Z","name":"ewfrfwe.xyz","record_type":"a"},"aaa.bbb.xn--9kqt69cipkc3t.xn--fiqs8s":{"record_type":"a","resolve_time":"2026-04-10T03:42:19Z","name":"aaa.bbb.xn--9kqt69cipkc3t.xn--fiqs8s"},"awin.ddns-ip.net":{"resolve_time":"2026-04-09T14:45:13Z","name":"awin.ddns-ip.net","record_type":"a"},"d.xj.mydeertrip.com":{"resolve_time":"2026-04-09T14:28:37Z","name":"d.xj.mydeertrip.com","record_type":"a"},"nxa.de5.net":{"record_type":"a","resolve_time":"2026-04-09T14:46:05Z","name":"nxa.de5.net"},"cliunou.yiyuyazhu.de5.net":{"resolve_time":"2026-04-09T14:45:10Z","name":"cliunou.yiyuyazhu.de5.net","record_type":"a"},"dev-medicapt.org":{"resolve_time":"2026-04-09T14:51:35Z","name":"dev-medicapt.org","record_type":"a"},"eyihz.com.cfd":{"resolve_time":"2026-04-09T17:17:39Z","name":"eyihz.com.cfd","record_type":"a"},"gossettmktg.com":{"resolve_time":"2026-04-09T14:31:56Z","name":"gossettmktg.com","record_type":"a"},"pcdn.ddns-ip.net":{"resolve_time":"2026-04-09T14:45:13Z","name":"pcdn.ddns-ip.net","record_type":"a"},"turabet727.com":{"resolve_time":"2026-04-09T14:33:33Z","name":"turabet727.com","record_type":"a"},"wmx.de5.net":{"resolve_time":"2026-04-09T14:45:28Z","name":"wmx.de5.net","record_type":"a"},"www.cotown.net":{"resolve_time":"2026-04-09T14:45:11Z","name":"www.cotown.net","record_type":"a"},"chat05.asdfhjkfg.casacam.net":{"resolve_time":"2026-04-09T14:45:36Z","name":"chat05.asdfhjkfg.casacam.net","record_type":"a"},"lamis.ma":{"resolve_time":"2026-04-09T14:43:54Z","name":"lamis.ma","record_type":"a"},"tknjsfd.gq":{"resolve_time":"2026-04-09T14:40:55Z","name":"tknjsfd.gq","record_type":"a"},"www.marcoshandgemaakteschoenen.nl":{"name":"www.marcoshandgemaakteschoenen.nl","record_type":"a","resolve_time":"2026-04-09T14:50:31Z"},"www.yangergou.eu.org":{"resolve_time":"2026-04-09T14:52:18Z","name":"www.yangergou.eu.org","record_type":"a"},"xn--tiq9z.xn--9kqt69cipkc3t.xn--fiqs8s":{"resolve_time":"2026-04-10T02:14:42Z","name":"xn--tiq9z.xn--9kqt69cipkc3t.xn--fiqs8s","record_type":"a"},"culturama.fun":{"resolve_time":"2026-04-10T00:25:53Z","name":"culturama.fun","record_type":"a"},"masterscursos.online":{"name":"masterscursos.online","record_type":"a","resolve_time":"2026-04-09T14:51:07Z"},"rhddyndns.com":{"resolve_time":"2026-04-09T14:34:25Z","name":"rhddyndns.com","record_type":"a"},"sheerr-market.com":{"resolve_time":"2026-04-09T14:31:49Z","name":"sheerr-market.com","record_type":"a"},"zzz.xn--9kqt69cipkc3t.xn--fiqs8s":{"resolve_time":"2026-04-10T03:56:29Z","name":"zzz.xn--9kqt69cipkc3t.xn--fiqs8s","record_type":"a"},"fle.ma":{"record_type":"a","resolve_time":"2026-04-09T14:43:50Z","name":"fle.ma"},"geocapital.ma":{"resolve_time":"2026-04-09T14:43:53Z","name":"geocapital.ma","record_type":"a"},"mybabysamples.com":{"resolve_time":"2026-04-09T14:28:38Z","name":"mybabysamples.com","record_type":"a"},"oldcraft.top":{"resolve_time":"2026-04-09T14:55:07Z","name":"oldcraft.top","record_type":"a"},"woxo.de5.net":{"record_type":"a","resolve_time":"2026-04-09T14:45:42Z","name":"woxo.de5.net"},"3000.gleeze.com":{"resolve_time":"2026-04-09T16:02:00Z","name":"3000.gleeze.com","record_type":"a"},"dinalconcept.com.ng":{"resolve_time":"2026-04-09T14:49:58Z","name":"dinalconcept.com.ng","record_type":"a"},"dns.diamondking.in":{"record_type":"a","resolve_time":"2026-04-09T14:42:25Z","name":"dns.diamondking.in"},"fcu.re.mw":{"resolve_time":"2026-04-09T17:06:10Z","name":"fcu.re.mw","record_type":"a"},"sdjc.oesoc.net":{"name":"sdjc.oesoc.net","record_type":"a","resolve_time":"2026-04-09T14:47:18Z"},"test.lblb.eu.org":{"name":"test.lblb.eu.org","record_type":"a","resolve_time":"2026-04-09T14:52:08Z"},"www.luxmarket.ma":{"record_type":"a","resolve_time":"2026-04-09T14:43:54Z","name":"www.luxmarket.ma"},"dingyue.kingyu520.ddns-ip.net":{"name":"dingyue.kingyu520.ddns-ip.net","record_type":"a","resolve_time":"2026-04-09T14:45:15Z"},"endm.ma":{"resolve_time":"2026-04-09T14:43:49Z","name":"endm.ma","record_type":"a"},"glitch.ma":{"resolve_time":"2026-04-09T14:43:53Z","name":"glitch.ma","record_type":"a"},"jklqwe.de5.net":{"resolve_time":"2026-04-09T14:45:21Z","name":"jklqwe.de5.net","record_type":"a"},"local.snapshot.jrkyushu.co.jp":{"resolve_time":"2026-04-09T14:42:51Z","name":"local.snapshot.jrkyushu.co.jp","record_type":"a"},"prod.bumastemra.salt.ws":{"name":"prod.bumastemra.salt.ws","record_type":"a","resolve_time":"2026-04-09T14:56:07Z"},"qqqq.ddns-ip.net":{"resolve_time":"2026-04-09T14:45:48Z","name":"qqqq.ddns-ip.net","record_type":"a"},"demech.net":{"resolve_time":"2026-04-09T14:46:01Z","name":"demech.net","record_type":"a"},"hanyc.ddns-ip.net":{"resolve_time":"2026-04-09T14:45:13Z","name":"hanyc.ddns-ip.net","record_type":"a"},"yang.com.cfd":{"record_type":"a","resolve_time":"2026-04-09T16:01:45Z","name":"yang.com.cfd"},"zkfnas.de5.net":{"resolve_time":"2026-04-09T14:45:16Z","name":"zkfnas.de5.net","record_type":"a"},"kxsw1.yangxiaoyong.eu.org":{"resolve_time":"2026-04-09T14:52:17Z","name":"kxsw1.yangxiaoyong.eu.org","record_type":"a"},"chiyi4488.eu.org":{"name":"chiyi4488.eu.org","record_type":"a","resolve_time":"2026-04-09T14:51:35Z"},"exx.ddns.net":{"resolve_time":"2026-04-09T14:46:05Z","name":"exx.ddns.net","record_type":"a"},"fcu8.com.cfd":{"resolve_time":"2026-04-09T17:06:56Z","name":"fcu8.com.cfd","record_type":"a"},"firewall.de5.net":{"resolve_time":"2026-04-09T14:45:39Z","name":"firewall.de5.net","record_type":"a"},"nanatest.xyz":{"record_type":"a","resolve_time":"2026-04-09T14:56:19Z","name":"nanatest.xyz"},"thinkspace.thinkloud.de":{"resolve_time":"2026-04-09T14:41:20Z","name":"thinkspace.thinkloud.de","record_type":"a"},"www.vonapartis.gr":{"resolve_time":"2026-04-09T14:42:51Z","name":"www.vonapartis.gr","record_type":"a"},"bbs.lblb.eu.org":{"resolve_time":"2026-04-09T14:52:05Z","name":"bbs.lblb.eu.org","record_type":"a"},"gortorg.com":{"resolve_time":"2026-04-09T14:31:49Z","name":"gortorg.com","record_type":"a"}}},"ip":"8.8.8.8","location":{"province":"California","coordinates":{"latitude":37.4056,"longitude":-122.0775},"continent":"North America","country":"United States","country_code":"US","city":"Mountain View","postal_code":"94043","timezone":"America/Los_Angeles"},"autonomous_system":{"name":"GOOGLE - Google LLC","country_code":"US","asn":15169,"description":"GOOGLE - Google LLC","bgp_prefix":"8.8.8.0/24"}},"extensions":{}}}}
````

## Input

### New-line Separated IPs
By default, ZAnnotate expects new-line delimited IP addresses on standard input. For example:
```shell
printf "1.1.1.1\n8.8.8.8" | zannotate --rdns
```

```jsonl
{"ip":"1.1.1.1","rdns":{"domain_names":["one.one.one.one"]}}
{"ip":"8.8.8.8","rdns":{"domain_names":["dns.google"]}}
```

### JSON and CSV Flags

The `--output-annotation-field` flag can be used to specify a different field name for the annotations instead of `zannotate` for both CSV and JSON. For example:

```shell
printf "name,ip_address,date\n cloudflare,1.1.1.1,04-04-26\n google,8.8.8.8,04-04-26" | ./zannotate --rdns --input-file-type=csv --input-ip-field=ip_address --output-annotation-field="info"
```

```json lines
{"name":" cloudflare","ip_address":"1.1.1.1","date":"04-04-26","info":{"rdns":{"domain_names":["one.one.one.one"]}}}
{"ip_address":"8.8.8.8","date":"04-04-26","info":{"rdns":{"domain_names":["dns.google"]}},"name":" google"}
```
### JSON
You may wish to annotate data that is already in JSON format. You'll then need to use the `--input-file-type=json` flag.
This will insert a `zannotate` field into the existing JSON object. For example:

```shell
echo '{"ip": "1.1.1.1"}'  | ./zannotate --rdns --geoasn --geoasn-database=/path-to-geo-asn.mmdb --input-file-type=json    
```

```json
{"ip":"1.1.1.1","zannotate":{"geoasn":{"asn":13335,"org":"CLOUDFLARENET"},"rdns":{"domain_names":["one.one.one.one"]}}}
```

If your JSON objects have a different field for the IP address, you can specify that with the `--input-ip-field` flag. For example, if your JSON objects have an `ip_address` field instead of `ip`, you can use:

```shell
echo '{"ip_address": "1.1.1.1"}'  | ./zannotate --rdns --input-file-type=json --input-ip-field=ip_address    
```

```json
{"ip_address":"1.1.1.1","zannotate":{"rdns":{"domain_names":["one.one.one.one"]}}}
````

### CSV
If your input data is in CSV format, you can use the `--input-file-type=csv` flag.

```shell
printf "name,ip,date\n cloudflare,1.1.1.1,04-04-26\n google,8.8.8.8,04-04-26" | ./zannotate --rdns --input-file-type=csv 
```

```jsonl
{"name":" cloudflare","ip":"1.1.1.1","date":"04-04-26","zannotate":{"rdns":{"domain_names":["one.one.one.one"]}}}
{"name":" google","ip":"8.8.8.8","date":"04-04-26","zannotate":{"rdns":{"domain_names":["dns.google"]}}}
```

Similar to JSON, you can use the `--input-ip-field` flag to specify a column other than `ip` that contains the IP address.

```shell
printf "name,ip_address,date\n cloudflare,1.1.1.1,04-04-26\n google,8.8.8.8,04-04-26" | ./zannotate --rdns --input-file-type=csv --input-ip-field=ip_address
```

```jsonl
{"date":"04-04-26","zannotate":{"rdns":{"domain_names":["dns.google"]}},"name":" google","ip_address":"8.8.8.8"}
{"date":"04-04-26","zannotate":{"rdns":{"domain_names":["one.one.one.one"]}},"name":" cloudflare","ip_address":"1.1.1.1"}
```



# Modules

## RDAP (WHOIS successor)
RDAP (Registration Data Access Protocol) is a protocol used to access registration data for internet resources such as 
IP addresses and domain names and is the successor to WHOIS. ZAnnotate can query RDAP servers to pull registration data for IPs.

```shell
echo "1.1.1.1" | ./zannotate --rdap 
```

```json
{"ip":"1.1.1.1","whois":{"DecodeData":{},"Lang":"","Conformance":["history_version_0","nro_rdap_profile_0","cidr0","rdap_level_0"],"ObjectClassName":"ip network","Notices":[{"DecodeData":{},"Title":"Source","Type":"","Description":["Objects returned came from source","APNIC"],"Links":null},{"DecodeData":{},"Title":"Terms and Conditions","Type":"","Description":["This is the APNIC WHOIS Database query service. The objects are in RDAP format."],"Links":[{"DecodeData":{},"Value":"https://rdap.apnic.net/ip/1.1.1.1","Rel":"terms-of-service","Href":"http://www.apnic.net/db/dbcopyright.html","HrefLang":null,"Title":"","Media":"","Type":"text/html"}]},{"DecodeData":{},"Title":"Whois Inaccuracy Reporting","Type":"","Description":["If you see inaccuracies in the results, please visit: "],"Links":[{"DecodeData":{},"Value":"https://rdap.apnic.net/ip/1.1.1.1","Rel":"inaccuracy-report","Href":"https://www.apnic.net/manage-ip/using-whois/abuse-and-spamming/invalid-contact-form","HrefLang":null,"Title":"","Media":"","Type":"text/html"}]}],"Handle":"1.1.1.0 - 1.1.1.255","StartAddress":"1.1.1.0","EndAddress":"1.1.1.255","IPVersion":"v4","Name":"APNIC-LABS","Type":"ASSIGNED PORTABLE","Country":"AU","ParentHandle":"","Status":["active"],"Entities":[{"DecodeData":{},"Lang":"","Conformance":null,"ObjectClassName":"entity","Notices":null,"Handle":"IRT-APNICRANDNET-AU","VCard":{"Properties":[{"Name":"version","Parameters":{},"Type":"text","Value":"4.0"},{"Name":"fn","Parameters":{},"Type":"text","Value":"IRT-APNICRANDNET-AU"},{"Name":"kind","Parameters":{},"Type":"text","Value":"group"},{"Name":"adr","Parameters":{"label":["PO Box 3646\nSouth Brisbane, QLD 4101\nAustralia"]},"Type":"text","Value":["","","","","","",""]},{"Name":"email","Parameters":{},"Type":"text","Value":"helpdesk@apnic.net"},{"Name":"email","Parameters":{"pref":["1"]},"Type":"text","Value":"helpdesk@apnic.net"}]},"Roles":["abuse"],"PublicIDs":null,"Entities":null,"Remarks":[{"DecodeData":{},"Title":"remarks","Type":"","Description":["helpdesk@apnic.net was validated on 2021-02-09"],"Links":null}],"Links":[{"DecodeData":{},"Value":"https://rdap.apnic.net/ip/1.1.1.1","Rel":"self","Href":"https://rdap.apnic.net/entity/IRT-APNICRANDNET-AU","HrefLang":null,"Title":"","Media":"","Type":"application/rdap+json"}],"Events":[{"DecodeData":{},"Action":"registration","Actor":"","Date":"2011-04-12T17:56:54Z","Links":null},{"DecodeData":{},"Action":"last changed","Actor":"","Date":"2025-11-18T00:26:57Z","Links":null}],"AsEventActor":null,"Status":null,"Port43":"","Networks":null,"Autnums":null},{"DecodeData":{},"Lang":"","Conformance":null,"ObjectClassName":"entity","Notices":null,"Handle":"ORG-ARAD1-AP","VCard":{"Properties":[{"Name":"version","Parameters":{},"Type":"text","Value":"4.0"},{"Name":"fn","Parameters":{},"Type":"text","Value":"APNIC Research and Development"},{"Name":"kind","Parameters":{},"Type":"text","Value":"org"},{"Name":"adr","Parameters":{"label":["6 Cordelia St"]},"Type":"text","Value":["","","","","","",""]},{"Name":"tel","Parameters":{"type":["voice"]},"Type":"text","Value":"+61-7-38583100"},{"Name":"tel","Parameters":{"type":["fax"]},"Type":"text","Value":"+61-7-38583199"},{"Name":"email","Parameters":{},"Type":"text","Value":"helpdesk@apnic.net"}]},"Roles":["registrant"],"PublicIDs":null,"Entities":null,"Remarks":null,"Links":[{"DecodeData":{},"Value":"https://rdap.apnic.net/ip/1.1.1.1","Rel":"self","Href":"https://rdap.apnic.net/entity/ORG-ARAD1-AP","HrefLang":null,"Title":"","Media":"","Type":"application/rdap+json"}],"Events":[{"DecodeData":{},"Action":"registration","Actor":"","Date":"2017-08-08T23:21:55Z","Links":null},{"DecodeData":{},"Action":"last changed","Actor":"","Date":"2023-09-05T02:15:19Z","Links":null}],"AsEventActor":null,"Status":null,"Port43":"","Networks":null,"Autnums":null},{"DecodeData":{},"Lang":"","Conformance":null,"ObjectClassName":"entity","Notices":null,"Handle":"AIC3-AP","VCard":{"Properties":[{"Name":"version","Parameters":{},"Type":"text","Value":"4.0"},{"Name":"fn","Parameters":{},"Type":"text","Value":"APNICRANDNET Infrastructure Contact"},{"Name":"kind","Parameters":{},"Type":"text","Value":"group"},{"Name":"adr","Parameters":{"label":["6 Cordelia St South Brisbane QLD 4101"]},"Type":"text","Value":["","","","","","",""]},{"Name":"tel","Parameters":{"type":["voice"]},"Type":"text","Value":"+61 7 3858 3100"},{"Name":"email","Parameters":{},"Type":"text","Value":"research@apnic.net"}]},"Roles":["administrative","technical"],"PublicIDs":null,"Entities":null,"Remarks":null,"Links":[{"DecodeData":{},"Value":"https://rdap.apnic.net/ip/1.1.1.1","Rel":"self","Href":"https://rdap.apnic.net/entity/AIC3-AP","HrefLang":null,"Title":"","Media":"","Type":"application/rdap+json"}],"Events":[{"DecodeData":{},"Action":"registration","Actor":"","Date":"2023-04-26T00:42:16Z","Links":null},{"DecodeData":{},"Action":"last changed","Actor":"","Date":"2024-07-18T04:37:37Z","Links":null}],"AsEventActor":null,"Status":null,"Port43":"","Networks":null,"Autnums":null}],"Remarks":[{"DecodeData":{},"Title":"description","Type":"","Description":["APNIC and Cloudflare DNS Resolver project","Routed globally by AS13335/Cloudflare","Research prefix for APNIC Labs"],"Links":null},{"DecodeData":{},"Title":"remarks","Type":"","Description":["---------------","All Cloudflare abuse reporting can be done via","resolver-abuse@cloudflare.com","---------------"],"Links":null}],"Links":[{"DecodeData":{},"Value":"https://rdap.apnic.net/ip/1.1.1.1","Rel":"self","Href":"https://rdap.apnic.net/ip/1.1.1.0/24","HrefLang":null,"Title":"","Media":"","Type":"application/rdap+json"}],"Port43":"whois.apnic.net","Events":[{"DecodeData":{},"Action":"registration","Actor":"","Date":"2011-08-10T23:12:35Z","Links":null},{"DecodeData":{},"Action":"last changed","Actor":"","Date":"2023-04-26T22:57:58Z","Links":null}]}}
```

This should give you the same output as a direct query to an RDAP server, for example:

```shell
rdap 1.1.1.1
```
```text
IP Network:
  Handle: 1.1.1.0 - 1.1.1.255
  Start Address: 1.1.1.0
  End Address: 1.1.1.255
  IP Version: v4
  Name: APNIC-LABS
  Type: ASSIGNED PORTABLE
...<further output truncated for brevity>
```
