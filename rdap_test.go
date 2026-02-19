package zannotate

import (
	"net"
	"testing"

	"github.com/openrdap/rdap"
)

func TestRDAPAnnotatorCloudflareSmokeTest(t *testing.T) {
	// Build factory and annotator
	factory := &RDAPAnnotatorFactory{Timeout: 2}
	a := factory.MakeAnnotator(0).(*RDAPAnnotator)

	res := a.Annotate(net.ParseIP("1.1.1.1"))
	if res == nil {
		t.Fatal("RDAPAnnotator failed to annotate 1.1.1.1")
	}

	ans := res.(*rdap.IPNetwork)
	if ans.Name != "APNIC-LABS" {
		t.Fatalf("RDAPAnnotator failed to annotate 1.1.1.1's name correctly, is owned by APNIC-LABS, got %s", ans.Name)
	}
}
