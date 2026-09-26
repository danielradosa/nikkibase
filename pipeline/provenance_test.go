package pipeline

import (
	"strings"
	"testing"
)

const registryJSON = `{"sources":[
 {"id":"wiki","name":"Wiki","url":"https://wiki.example","flags":["fandom"],"licence":"CC BY-SA 3.0","licenceEvidence":"licensing page","redistribution":"permitted","status":"redistributable","fields":["grades"]},
 {"id":"calc","name":"Calc","url":"https://calc.example","flags":["names","keys"],"licence":"none located","redistribution":"not established","status":"permission-required","fields":["names"]},
 {"id":"ids","name":"IDs","url":"","flags":["known"],"licence":"unknown","redistribution":"unknown","status":"unknown","fields":["ids"]}
]}`

func registry(t *testing.T) *Registry {
	t.Helper()
	r, err := ReadRegistry([]byte(registryJSON))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func source(t *testing.T, r *Registry, flag string) Source {
	t.Helper()
	s, ok := r.ForFlag(flag)
	if !ok {
		t.Fatalf("no source for -%s", flag)
	}
	return s
}

func TestGateRefusesUnlicensedByName(t *testing.T) {
	r := registry(t)
	_, err := Gate([]Source{source(t, r, "fandom"), source(t, r, "names")}, nil, false)
	if err == nil {
		t.Fatal("a permission-required source with no exception was admitted")
	}
	if !strings.Contains(err.Error(), "Calc (calc") || !strings.Contains(err.Error(), "-allow-unlicensed") {
		t.Errorf("error does not name the blocked source and the way out: %v", err)
	}
}

func TestGateAdmitsLicensedAlone(t *testing.T) {
	r := registry(t)
	basis, err := Gate([]Source{source(t, r, "fandom")}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if basis["wiki"] != ByLicence {
		t.Errorf("basis = %q, want %q", basis["wiki"], ByLicence)
	}
}

func TestGateAdmitsRecordedException(t *testing.T) {
	r := registry(t)
	ex, err := ReadExceptions([]byte(`{"exceptions":[{"source":"calc","decidedBy":"D","decidedOn":"2026-09-23","reason":"names"}]}`), r)
	if err != nil {
		t.Fatal(err)
	}
	basis, err := Gate([]Source{source(t, r, "names")}, ex, false)
	if err != nil {
		t.Fatal(err)
	}
	if basis["calc"] != ByException {
		t.Errorf("basis = %q, want %q", basis["calc"], ByException)
	}
	if _, err := Gate([]Source{source(t, r, "known")}, ex, false); err == nil {
		t.Error("an unknown-status source was admitted on the strength of another source's exception")
	}
}

func TestExceptionsCannotCoverOtherStatuses(t *testing.T) {
	r := registry(t)
	for _, id := range []string{"wiki", "ids", "nope"} {
		_, err := ReadExceptions([]byte(`{"exceptions":[{"source":"`+id+`","decidedBy":"D","decidedOn":"x","reason":"y"}]}`), r)
		if err == nil {
			t.Errorf("an exception for %q was accepted", id)
		}
	}
	if _, err := ReadExceptions([]byte(`{"exceptions":[{"source":"calc"}]}`), r); err == nil {
		t.Error("an exception with no decider, date or reason was accepted")
	}
}

func TestRegistryRefusesRedistributableWithoutEvidence(t *testing.T) {
	_, err := ReadRegistry([]byte(`{"sources":[{"id":"x","flags":["a"],"licence":"MIT","status":"redistributable"}]}`))
	if err == nil {
		t.Error("a source was marked redistributable with no licence evidence")
	}
}

func TestAllowUnlicensedMakesResearchBundle(t *testing.T) {
	r := registry(t)
	basis, err := Gate([]Source{source(t, r, "names")}, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	p := NewProvenance("v", "t", r, nil, basis, nil, true)
	if p.Mode != "research" {
		t.Errorf("mode = %q, want research", p.Mode)
	}
	if err := CheckDeployable(p, r, nil); err == nil {
		t.Error("a research bundle passed the deploy check")
	}
}

func TestProvenanceAndRevocation(t *testing.T) {
	r := registry(t)
	ex, _ := ReadExceptions([]byte(`{"exceptions":[{"source":"calc","decidedBy":"D","decidedOn":"2026-09-23","reason":"names"}]}`), r)
	used := []Source{source(t, r, "fandom"), source(t, r, "names")}
	basis, err := Gate(used, ex, false)
	if err != nil {
		t.Fatal(err)
	}
	p := NewProvenance("v1", "2026-09-23T00:00:00Z", r, ex, basis, map[string][]Input{"wiki": {{File: "dump.xml", SHA256: "ab", Bytes: 1}}}, false)
	if len(p.Sources) != len(r.Sources) {
		t.Fatalf("provenance lists %d sources, registry has %d", len(p.Sources), len(r.Sources))
	}
	got := map[string]SourceRecord{}
	for _, s := range p.Sources {
		got[s.ID] = s
	}
	if !got["wiki"].Included || got["wiki"].Basis != ByLicence || len(got["wiki"].Inputs) != 1 {
		t.Errorf("wiki record = %+v", got["wiki"])
	}
	if !got["calc"].Included || got["calc"].Basis != ByException || got["calc"].Exception == nil {
		t.Errorf("calc record = %+v", got["calc"])
	}
	if got["ids"].Included || !strings.Contains(got["ids"].Summary, "not included in production") {
		t.Errorf("ids record = %+v", got["ids"])
	}
	if err := CheckDeployable(p, r, ex); err != nil {
		t.Errorf("a bundle built under a standing exception failed the deploy check: %v", err)
	}
	if err := CheckDeployable(p, r, nil); err == nil {
		t.Error("the deploy check passed after the exception was withdrawn")
	}
}

const permittedJSON = `{"sources":[
 {"id":"calc","name":"Calc","url":"https://calc.example","flags":["names"],"licence":"none","redistribution":"permitted in writing","status":"permitted","permission":"a free site, with credit","permissionUrl":"https://calc.example/reply","attributionRequired":true,"attribution":"Calc","fields":["names"]}
]}`

func TestPermittedSourceShipsOnItsPermission(t *testing.T) {
	r, err := ReadRegistry([]byte(permittedJSON))
	if err != nil {
		t.Fatal(err)
	}
	basis, err := Gate([]Source{source(t, r, "names")}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if basis["calc"] != ByPermission {
		t.Errorf("basis = %q, want %q", basis["calc"], ByPermission)
	}
	p := NewProvenance("v", "t", r, nil, basis, nil, false)
	rec := p.Sources[0]
	if !rec.Included || rec.Basis != ByPermission || rec.Exception != nil || !strings.Contains(rec.Summary, "a free site, with credit") {
		t.Errorf("calc record = %+v", rec)
	}
	if err := CheckDeployable(p, r, nil); err != nil {
		t.Errorf("a bundle built on a standing permission failed the deploy check: %v", err)
	}
	if _, err := ReadExceptions([]byte(`{"exceptions":[{"source":"calc","decidedBy":"D","decidedOn":"x","reason":"y"}]}`), r); err == nil {
		t.Error("an exception was accepted for a permitted source")
	}
	withdrawn, err := ReadRegistry([]byte(strings.Replace(permittedJSON, `"status":"permitted"`, `"status":"permission-required"`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckDeployable(p, withdrawn, nil); err == nil {
		t.Error("the deploy check passed after the permission was withdrawn")
	}
}

func TestRegistryRefusesPermittedWithoutEvidence(t *testing.T) {
	for _, s := range []string{
		`{"id":"x","flags":["a"],"licence":"none","status":"permitted","permission":"a free site"}`,
		`{"id":"x","flags":["a"],"licence":"none","status":"permitted","permissionUrl":"https://x.example"}`,
	} {
		if _, err := ReadRegistry([]byte(`{"sources":[` + s + `]}`)); err == nil {
			t.Errorf("a permitted source with its permission half on record was accepted: %s", s)
		}
	}
}
