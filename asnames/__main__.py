import json
import urllib2
import re

class CIDRReportASNameDump(object):
    """ASNNameDump represents the full list of
    registered AS Numbers on CIDR report. Acts as an iterator that
    will provide tuples: (AS Number, AS Name)"""

    CIDR_REPORT_URL = "http://www.cidr-report.org/as2.0/autnums.html"
    ENTRY_REGEX = re.compile("^<a href.*>(.*)</a>(.*)$")

    OVERRIDES = {
    }

    ADDITIONAL_NAMES = {
    }

    def __init__(self):
        self.__f = None
        self.data = {}

    def fetch(self):
        if not self.__f:
            self.__f = urllib2.urlopen(self.CIDR_REPORT_URL)
        for line in self.__f.readlines():
            m = self.ENTRY_REGEX.match(line)
            if m:
                asn = m.groups()[0].rstrip().lstrip().replace('AS','')
                # handle weird . notation for > 16-bit ASNs
                if '.' in asn:
                    b, s = asn.split('.')
                    asn = (int(b) << 16) + int(s)
                else:
                    asn = int(asn)
                description = m.groups()[1].rstrip().lstrip()
                # parse country code if available
                if len(description) > 3 and description[-3] == ",":
                    id_org = description[:-3]
                    country = description[-2:]
                elif len(description) > 4 and description[-4] == ",":
                    id_org = description[:-4].strip()
                    country = description[-3:].strip()
                else: # no country present.
                    id_org = description
                    country = None
                # this format is terrible, but we'll try to parse it out anyway.
                if " - " in id_org and id_org.split(" - ", 1)[0].isupper():
                    name, org = id_org.split(" - ", 1)
                #elif " " in id_org and id_org.split(" ", 1)[0].isupper():
                #    name, org = id_org.split(" ", 1)
                elif "-" in id_org and id_org.split("-", 1)[0].isupper():
                    name, org = id_org.split("-", 1)
                else:
                    name, org = id_org, None
                if name:
                    name = unicode(name, errors="ignore").encode("utf-8", "ignore").replace('"', '')
                if country:
                    country = unicode(country, errors="ignore").encode("utf-8", "ignore").replace('"', '')
                if org:
                    org = unicode(org, errors="ignore").encode("utf-8", "ignore").replace('"', '')
                if description:
                    description = unicode(description, errors="ignore").encode("utf-8", "ignore").replace('"', '')
                self.data[int(asn)] = {"asn":int(asn), "description":description, "country_code":country, "organization":org, "name":name}

    def lookup(self, number):
        number = int(number)
        if number in self.OVERRIDES:
            return self.OVERRIDES[number]
        if number in self.data:
            return self.data[number]
        if number in self.ADDITIONAL_NAMES:
            return self.ADDITIONAL_NAMES[number]
        return {"asn":number, "name":"UNKNOWN-%i" % number, "description":"Unknown AS (ASN:%i)" % number, "organization":"Unknown"}

    def iter(self):
        for asn, info in self.data.iteritems():
            yield info


def main():
    db = CIDRReportASNameDump()
    db.fetch()
    for r in db.iter():
        print json.dumps(r)

if __name__ == "__main__":
    main()
