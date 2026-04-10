package zannotate

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
)

// The full JSON response captured from a real API call to Censys for 1.1.1.1
const censysMockResponse = `
{
  "result": {
    "resource": {
      "ip": "1.1.1.1",
      "location": {
        "continent": "Oceania",
        "country": "Australia",
        "country_code": "AU",
        "city": "Brisbane",
        "postal_code": "9010",
        "timezone": "Australia/Brisbane",
        "province": "Queensland",
        "coordinates": {
          "latitude": -27.4679,
          "longitude": 153.0281
        }
      },
      "autonomous_system": {
        "asn": 13335,
        "description": "CLOUDFLARENET - Cloudflare, Inc.",
        "bgp_prefix": "1.1.1.0/24",
        "name": "CLOUDFLARENET - Cloudflare, Inc.",
        "country_code": "US"
      },
      "whois": {
        "network": {
          "handle": "APNIC-LABS",
          "name": "APNIC and Cloudflare DNS Resolver project",
          "cidrs": [
            "1.1.1.0/24"
          ],
          "updated": "2023-04-26T00:00:00Z"
        },
        "organization": {
          "handle": "ORG-ARAD1-AP",
          "name": "APNIC Research and Development",
          "address": "********",
          "country": "AU"
        }
      },
      "services": [
        {
          "port": 53,
          "protocol": "DNS",
          "transport_protocol": "udp",
          "ip": "1.1.1.1",
          "scan_time": "2026-04-09T04:34:30Z",
          "dns": {
            "server_type": "forwarding",
            "resolves_correctly": true,
            "answers": [
              {
                "name": "ip.parrotdns.com.",
                "response": "172.70.129.239",
                "type": "a"
              },
              {
                "name": "ip.parrotdns.com.",
                "response": "35.202.119.40",
                "type": "a"
              }
            ],
            "questions": [
              {
                "name": "ip.parrotdns.com.",
                "response": ";ip.parrotdns.com.\tIN\t A",
                "type": "a"
              }
            ],
            "edns": {
              "do": true,
              "udp": 1232
            },
            "r_code": "success"
          }
        },
        {
          "port": 80,
          "protocol": "HTTP",
          "transport_protocol": "tcp",
          "software": [
            {
              "vendor": "cloudflare",
              "product": "cloudflare_load_balancer"
            },
            {
              "type": [
                "WAF"
              ],
              "vendor": "cloudflare",
              "product": "waf"
            }
          ],
          "labels": [
            {
              "value": "WAF"
            }
          ],
          "ip": "1.1.1.1",
          "scan_time": "2026-04-10T00:44:26Z",
          "endpoints": [
            {
              "hostname": "1.1.1.1",
              "port": 80,
              "path": "/",
              "endpoint_type": "HTTP",
              "transport_protocol": "tcp",
              "scan_time": "2026-04-10T00:44:26Z",
              "http": {
                "uri": "http://1.1.1.1/",
                "protocol": "HTTP/1.1",
                "status_code": 301,
                "status_reason": "Moved Permanently",
                "headers": {
                  "CF-RAY": {
                    "headers": [
                      "9e9db9191b3386fe-ORD"
                    ]
                  },
                  "Connection": {
                    "headers": [
                      "keep-alive"
                    ]
                  },
                  "Content-Length": {
                    "headers": [
                      "167"
                    ]
                  },
                  "Content-Type": {
                    "headers": [
                      "text/html"
                    ]
                  },
                  "Date": {
                    "headers": [
                      "<REDACTED>"
                    ]
                  },
                  "Location": {
                    "headers": [
                      "https://1.1.1.1/"
                    ]
                  },
                  "Server": {
                    "headers": [
                      "cloudflare"
                    ]
                  }
                },
                "html_tags": [
                  "<title>301 Moved Permanently</title>"
                ],
                "body_size": 167,
                "body": "<html>\r\n<head><title>301 Moved Permanently</title></head>\r\n<body>\r\n<center><h1>301 Moved Permanently</h1></center>\r\n<hr><center>cloudflare</center>\r\n</body>\r\n</html>\r\n",
                "html_title": "301 Moved Permanently",
                "body_hash_sha256": "446a6087825fa73eadb045e5a2e9e2adf7df241b571228187728191d961dda1f",
                "body_hash_sha1": "7436e0b4b1f8c222c38069890b75fa2baf9ca620",
                "supported_versions": [
                  "HTTP/1.1"
                ]
              },
              "ip": "1.1.1.1"
            }
          ]
        },
        {
          "port": 443,
          "protocol": "UNKNOWN",
          "transport_protocol": "quic",
          "ip": "1.1.1.1",
          "scan_time": "2026-04-09T21:33:12Z"
        },
        {
          "port": 443,
          "protocol": "HTTP",
          "transport_protocol": "tcp",
          "cert": {
            "fingerprint_sha256": "e3b02826789d653d224d3edacbe4e877cb7286fc4c922672f6226741ca57ad65",
            "fingerprint_sha1": "f88635017260d40b9eb417bee73737911b630e59",
            "fingerprint_md5": "3f5657c103693bc68bc35429e987b3d4",
            "tbs_fingerprint_sha256": "57adbfff93f66ad39303530e6dba7fc64d82a644e3755b2a7943500c5ffcc9d6",
            "tbs_no_ct_fingerprint_sha256": "e2697bf79dddcdbff679bb0ef20e847e11cb17a5004e41ced16815c90badddfc",
            "spki_fingerprint_sha256": "fe4bbbcf3cc2001a1df63241958fbf9830d5d1252d84a30925801db709f1525f",
            "parent_spki_fingerprint_sha256": "41e1b77957dff8f16f8a11a0e1498d82bfa656e6ce631255c4b89826142da4b4",
            "parsed": {
              "version": 3,
              "serial_number": "104760816198390176145736269975424776716",
              "issuer_dn": "C=US, ST=Texas, L=Houston, O=SSL Corp, CN=SSL.com SSL Intermediate CA ECC R2",
              "issuer": {
                "common_name": [
                  "SSL.com SSL Intermediate CA ECC R2"
                ],
                "country": [
                  "US"
                ],
                "locality": [
                  "Houston"
                ],
                "province": [
                  "Texas"
                ],
                "organization": [
                  "SSL Corp"
                ]
              },
              "subject_dn": "C=US, ST=California, L=San Francisco, O=Cloudflare\\, Inc., CN=cloudflare-dns.com",
              "subject": {
                "common_name": [
                  "cloudflare-dns.com"
                ],
                "country": [
                  "US"
                ],
                "locality": [
                  "San Francisco"
                ],
                "province": [
                  "California"
                ],
                "organization": [
                  "Cloudflare, Inc."
                ]
              },
              "subject_key_info": {
                "key_algorithm": {
                  "name": "ECDSA",
                  "oid": "1.2.840.10045.2.1"
                },
                "ecdsa": {
                  "b": "5ac635d8aa3a93e7b3ebbd55769886bc651d06b0cc53b0f63bce3c3e27d2604b",
                  "curve": "P-256",
                  "gx": "6b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296",
                  "gy": "4fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5",
                  "length": 256,
                  "n": "ffffffff00000000ffffffffffffffffbce6faada7179e84f3b9cac2fc632551",
                  "p": "ffffffff00000001000000000000000000000000ffffffffffffffffffffffff",
                  "pub": "046383502512ea727819eb3247afc105529c2a2b608a844e756d814847c1c7becf85796c12295b50b3ccec50a1949edc4408070c801a93d3bd78117bb6a3c8eaac",
                  "x": "6383502512ea727819eb3247afc105529c2a2b608a844e756d814847c1c7becf",
                  "y": "85796c12295b50b3ccec50a1949edc4408070c801a93d3bd78117bb6a3c8eaac"
                },
                "fingerprint_sha256": "96d43a697cb7b6aa4d64a25d9debcc0fba11f88b08e6b3566ceb2c143ae5f84c"
              },
              "validity_period": {
                "not_before": "2025-12-31T19:20:01Z",
                "not_after": "2026-12-21T19:20:01Z",
                "length_seconds": 30672001
              },
              "signature": {
                "signature_algorithm": {
                  "name": "ECDSA-SHA384",
                  "oid": "1.2.840.10045.4.3.3"
                },
                "value": "306402301b2eb53f7f34ee2a79c9dc5e3fe15aeaf3fd0581b24ec6cab641ef5480d4fed03010e89c5a727e41105a889600d7cf0f023012fce5ba42cf30d3c2296380704acb379151ea1e24a8c1337752ea4e3bb1e2348d5d6cc2b205639cec499f8ab7323285",
                "valid": true
              },
              "serial_number_hex": "4ed03304c46b87a8c2eb5569db9eba0c"
            },
            "names": [
              "*.cloudflare-dns.com",
              "1.0.0.1",
              "1.1.1.1",
              "162.159.36.1",
              "162.159.46.1",
              "cloudflare-dns.com",
              "one.one.one.one"
            ],
            "validation_level": "ov",
            "validation": {
              "nss": {
                "is_valid": true,
                "ever_valid": true,
                "has_trusted_path": true,
                "had_trusted_path": true,
                "chains": [
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "3417bb06cc6007da1b961c920b8ab4ce3fad820e4aa30b9acbc4a74ebdcebc65"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "06b9722a699c57dff1869f430b479bb6eb49aae1184eac9c5325c1334a34ea4c",
                      "85666a562ee0be5ce925c1d8890a6f76a87ec16d4d7d5f29ea7419cf20123b69"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "3f47228bd9ac6738f87a806336673127ce2e3da0dffa0793355ee2efff203b9e",
                      "cecddc905099d8dadfc5b1d209b737cbe2c18cfb2c10c0ff0bcf0d3286fc1aa2"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "a272f56f844fd9f8979fff8c3cb18621454cb15ba419fc4427824b7bcd2d29b6",
                      "f1c1b50ae5a20dd8030ec9f6bc24823dd367b5255759b4e71b61fce9f7375d73"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "06b9722a699c57dff1869f430b479bb6eb49aae1184eac9c5325c1334a34ea4c",
                      "3f7befce41f251b2282018fb73213a2116a3f4f692389335e04ff5ae33ab45da",
                      "cecddc905099d8dadfc5b1d209b737cbe2c18cfb2c10c0ff0bcf0d3286fc1aa2"
                    ]
                  }
                ],
                "parents": [
                  "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39"
                ],
                "type": "leaf"
              },
              "microsoft": {
                "is_valid": true,
                "ever_valid": true,
                "has_trusted_path": true,
                "had_trusted_path": true,
                "chains": [
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "3417bb06cc6007da1b961c920b8ab4ce3fad820e4aa30b9acbc4a74ebdcebc65"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "06b9722a699c57dff1869f430b479bb6eb49aae1184eac9c5325c1334a34ea4c",
                      "85666a562ee0be5ce925c1d8890a6f76a87ec16d4d7d5f29ea7419cf20123b69"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "3f47228bd9ac6738f87a806336673127ce2e3da0dffa0793355ee2efff203b9e",
                      "cecddc905099d8dadfc5b1d209b737cbe2c18cfb2c10c0ff0bcf0d3286fc1aa2"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "a272f56f844fd9f8979fff8c3cb18621454cb15ba419fc4427824b7bcd2d29b6",
                      "f1c1b50ae5a20dd8030ec9f6bc24823dd367b5255759b4e71b61fce9f7375d73"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "06b9722a699c57dff1869f430b479bb6eb49aae1184eac9c5325c1334a34ea4c",
                      "3f7befce41f251b2282018fb73213a2116a3f4f692389335e04ff5ae33ab45da",
                      "cecddc905099d8dadfc5b1d209b737cbe2c18cfb2c10c0ff0bcf0d3286fc1aa2"
                    ]
                  }
                ],
                "parents": [
                  "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39"
                ],
                "type": "leaf"
              },
              "apple": {
                "is_valid": true,
                "ever_valid": true,
                "has_trusted_path": true,
                "had_trusted_path": true,
                "chains": [
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "3417bb06cc6007da1b961c920b8ab4ce3fad820e4aa30b9acbc4a74ebdcebc65"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "06b9722a699c57dff1869f430b479bb6eb49aae1184eac9c5325c1334a34ea4c",
                      "85666a562ee0be5ce925c1d8890a6f76a87ec16d4d7d5f29ea7419cf20123b69"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "3f47228bd9ac6738f87a806336673127ce2e3da0dffa0793355ee2efff203b9e",
                      "cecddc905099d8dadfc5b1d209b737cbe2c18cfb2c10c0ff0bcf0d3286fc1aa2"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "a272f56f844fd9f8979fff8c3cb18621454cb15ba419fc4427824b7bcd2d29b6",
                      "f1c1b50ae5a20dd8030ec9f6bc24823dd367b5255759b4e71b61fce9f7375d73"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "06b9722a699c57dff1869f430b479bb6eb49aae1184eac9c5325c1334a34ea4c",
                      "3f7befce41f251b2282018fb73213a2116a3f4f692389335e04ff5ae33ab45da",
                      "cecddc905099d8dadfc5b1d209b737cbe2c18cfb2c10c0ff0bcf0d3286fc1aa2"
                    ]
                  }
                ],
                "parents": [
                  "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39"
                ],
                "type": "leaf"
              },
              "chrome": {
                "is_valid": true,
                "ever_valid": true,
                "has_trusted_path": true,
                "had_trusted_path": true,
                "chains": [
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "3417bb06cc6007da1b961c920b8ab4ce3fad820e4aa30b9acbc4a74ebdcebc65"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "06b9722a699c57dff1869f430b479bb6eb49aae1184eac9c5325c1334a34ea4c",
                      "85666a562ee0be5ce925c1d8890a6f76a87ec16d4d7d5f29ea7419cf20123b69"
                    ]
                  },
                  {
                    "sha256fp": [
                      "948b7111af42f546d579cff5ce2bdec82134dd9914842bddb0c52872eb604e39",
                      "a272f56f844fd9f8979fff8c3cb18621454cb15ba419fc4427824b7bcd2d29b6",
                      "f1c1b50ae5a20dd8030ec9f6bc24823dd367b5255759b4e71b61fce9f7375d73"
                    ]
                  }
                ]
              }
            }
          }
        }
      ]
    }
  }
}`

func newMockCensysServer(t *testing.T, expectedIP string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request path contains the IP
		expectedPath := "/v3/global/asset/host/" + expectedIP
		if r.URL.Path != expectedPath {
			t.Errorf("unexpected path: got %s, want %s", r.URL.Path, expectedPath)
		}
		// Verify auth header is present
		if r.Header.Get("authorization") == "" {
			t.Error("missing authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(censysMockResponse))
	}))
}

func newTestAnnotator(serverURL string) *CensysAnnotator {
	factory := &CensysAnnotatorFactory{
		client:        &http.Client{},
		personalToken: "test-token",
	}
	factory.Enabled = true
	factory.Threads = 1
	return &CensysAnnotator{
		Factory: factory,
		Id:      0,
	}
}

func TestAnnotate_HostLookup(t *testing.T) {
	ip := "1.1.1.1"
	server := newMockCensysServer(t, ip)
	defer server.Close()

	annotator := newTestAnnotator(server.URL)
	// Point the annotator at the mock server instead of the real API
	origURL := censysAPIHostLookupURL
	censysAPIHostLookupURL = server.URL + "/v3/global/asset/host/"
	defer func() { censysAPIHostLookupURL = origURL }()

	result := annotator.Annotate(net.ParseIP(ip))
	if result == nil {
		t.Fatal("expected non-nil result, got nil")
	}

	assert.Equal(t, "1.1.1.1", result.(map[string]any)["result"].(map[string]any)["resource"].(map[string]any)["ip"])
	assert.Equal(t, "Oceania", result.(map[string]any)["result"].(map[string]any)["resource"].(map[string]any)["location"].(map[string]any)["continent"])
	assert.Equal(t, "DNS", result.(map[string]any)["result"].(map[string]any)["resource"].(map[string]any)["services"].([]any)[0].(map[string]any)["protocol"])
}

func TestAnnotate_NilOnError(t *testing.T) {
	// Server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	annotator := newTestAnnotator(server.URL)
	origURL := censysAPIHostLookupURL
	censysAPIHostLookupURL = server.URL + "/v3/global/asset/host/"
	defer func() { censysAPIHostLookupURL = origURL }()

	result := annotator.Annotate(net.ParseIP("1.1.1.1"))
	if result != nil {
		t.Errorf("expected nil on error, got %v", result)
	}
}

func BenchmarkAnnotate_HostLookup(b *testing.B) {
	ip := net.ParseIP("1.1.1.1")
	t := &testing.T{}
	server := newMockCensysServer(t, ip.String())
	defer server.Close()

	annotator := newTestAnnotator(server.URL)
	// Point the annotator at the mock server instead of the real API
	origURL := censysAPIHostLookupURL
	censysAPIHostLookupURL = server.URL + "/v3/global/asset/host/"
	defer func() { censysAPIHostLookupURL = origURL }()


	b.ResetTimer()
	b.ReportAllocs()
	for _ = range b.N {
		result := annotator.Annotate(ip)
		if result == nil {
			b.Fatal("expected non-nil result, got nil")
		}
	}
}
