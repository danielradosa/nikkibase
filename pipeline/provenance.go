package pipeline

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type Status string

const (
	Redistributable    Status = "redistributable"
	Permitted          Status = "permitted"
	PermissionRequired Status = "permission-required"
	ResearchOnly       Status = "research-only"
	Proprietary        Status = "proprietary"
	Unknown            Status = "unknown"
)

type Source struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	By                  string   `json:"by,omitempty"`
	URL                 string   `json:"url"`
	Flags               []string `json:"flags"`
	Licence             string   `json:"licence"`
	LicenceURL          string   `json:"licenceUrl,omitempty"`
	LicenceEvidence     string   `json:"licenceEvidence,omitempty"`
	Permission          string   `json:"permission,omitempty"`
	PermissionURL       string   `json:"permissionUrl,omitempty"`
	Redistribution      string   `json:"redistribution"`
	Status              Status   `json:"status"`
	AttributionRequired bool     `json:"attributionRequired"`
	Attribution         string   `json:"attribution"`
	Fields              []string `json:"fields"`
}

type Registry struct {
	Note    string   `json:"note"`
	Sources []Source `json:"sources"`
}

type Exception struct {
	Source    string `json:"source"`
	DecidedBy string `json:"decidedBy"`
	DecidedOn string `json:"decidedOn"`
	Reason    string `json:"reason"`
}

type Exceptions struct {
	Note       string      `json:"note"`
	Exceptions []Exception `json:"exceptions"`
}

func ReadRegistry(b []byte) (*Registry, error) {
	var r Registry
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("sources: %w", err)
	}
	ids, flags := map[string]bool{}, map[string]string{}
	for _, s := range r.Sources {
		if s.ID == "" || ids[s.ID] {
			return nil, fmt.Errorf("sources: missing or duplicate id %q", s.ID)
		}
		ids[s.ID] = true
		switch s.Status {
		case Redistributable, Permitted, PermissionRequired, ResearchOnly, Proprietary, Unknown:
		default:
			return nil, fmt.Errorf("sources: %s has unknown status %q", s.ID, s.Status)
		}
		if s.Status == Redistributable && (s.Licence == "" || s.LicenceEvidence == "") {
			return nil, fmt.Errorf("sources: %s is marked redistributable with no licence evidence on record", s.ID)
		}
		if s.Status == Permitted && (s.Permission == "" || s.PermissionURL == "") {
			return nil, fmt.Errorf("sources: %s is marked permitted with no written permission on record", s.ID)
		}
		for _, f := range s.Flags {
			if other, dup := flags[f]; dup {
				return nil, fmt.Errorf("sources: flag -%s claimed by both %s and %s", f, other, s.ID)
			}
			flags[f] = s.ID
		}
	}
	return &r, nil
}

func (r *Registry) ForFlag(flag string) (Source, bool) {
	for _, s := range r.Sources {
		if slices.Contains(s.Flags, flag) {
			return s, true
		}
	}
	return Source{}, false
}

func ReadExceptions(b []byte, reg *Registry) (*Exceptions, error) {
	var e Exceptions
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, fmt.Errorf("exceptions: %w", err)
	}
	for _, x := range e.Exceptions {
		i := slices.IndexFunc(reg.Sources, func(s Source) bool { return s.ID == x.Source })
		if i < 0 {
			return nil, fmt.Errorf("exceptions: %q is not a registered source", x.Source)
		}
		if st := reg.Sources[i].Status; st != PermissionRequired {
			return nil, fmt.Errorf("exceptions: %s is %s; only permission-required sources can be excepted", x.Source, st)
		}
		if x.DecidedBy == "" || x.DecidedOn == "" || x.Reason == "" {
			return nil, fmt.Errorf("exceptions: %s must say who decided, when and why", x.Source)
		}
	}
	return &e, nil
}

func (e *Exceptions) find(id string) *Exception {
	if e == nil {
		return nil
	}
	for i := range e.Exceptions {
		if e.Exceptions[i].Source == id {
			return &e.Exceptions[i]
		}
	}
	return nil
}

type Basis string

const (
	ByLicence    Basis = "licence"
	ByPermission Basis = "permission"
	ByException  Basis = "recorded-exception"
	ForResearch  Basis = "research-only"
)

func Gate(used []Source, ex *Exceptions, allowUnlicensed bool) (map[string]Basis, error) {
	basis := map[string]Basis{}
	var blocked []string
	for _, s := range used {
		switch {
		case s.Status == Redistributable:
			basis[s.ID] = ByLicence
		case s.Status == Permitted:
			basis[s.ID] = ByPermission
		case s.Status == PermissionRequired && ex.find(s.ID) != nil:
			basis[s.ID] = ByException
		case allowUnlicensed:
			basis[s.ID] = ForResearch
		default:
			blocked = append(blocked, fmt.Sprintf("  %s (%s, via -%s): status %s", s.Name, s.ID, strings.Join(s.Flags, "/-"), s.Status))
		}
	}
	if len(blocked) > 0 {
		return nil, fmt.Errorf("refusing to build: these sources have no licence, written permission or recorded exception permitting production use:\n%s\n"+
			"Either drop them, record a deliberate exception in data/production-exceptions.json, "+
			"or pass -allow-unlicensed to build a research-only bundle that cannot be deployed", strings.Join(blocked, "\n"))
	}
	return basis, nil
}

type Input struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type SourceRecord struct {
	Source
	Included  bool       `json:"included"`
	Basis     Basis      `json:"basis,omitempty"`
	Exception *Exception `json:"exception,omitempty"`
	Inputs    []Input    `json:"inputs,omitempty"`
	Summary   string     `json:"summary"`
}

type Provenance struct {
	Schema  int            `json:"schema"`
	Version string         `json:"version"`
	BuiltAt string         `json:"builtAt"`
	Mode    string         `json:"mode"`
	Note    string         `json:"note"`
	Sources []SourceRecord `json:"sources"`
}

func NewProvenance(version, builtAt string, reg *Registry, ex *Exceptions, basis map[string]Basis,
	inputs map[string][]Input, allowUnlicensed bool) Provenance {
	p := Provenance{Schema: 1, Version: version, BuiltAt: builtAt, Mode: "production",
		Note: "Which sources this bundle was built from, and on what terms. Only sources with basis 'licence' are covered by a licence; 'permission' means the maintainer permitted the use in writing, on the terms recorded with the source; 'recorded-exception' means the data ships without established permission, by a dated decision recorded in data/production-exceptions.json."}
	if allowUnlicensed {
		p.Mode = "research"
		p.Note = "RESEARCH BUNDLE: built with -allow-unlicensed. It may include data with no established permission and must not be deployed."
	}
	for _, s := range reg.Sources {
		rec := SourceRecord{Source: s, Inputs: inputs[s.ID]}
		b, used := basis[s.ID]
		switch {
		case !used:
			rec.Summary = "Not used in this bundle; not included in production data."
		case b == ByLicence:
			rec.Included, rec.Basis = true, b
			rec.Summary = "Included under its licence: " + s.Licence + "."
		case b == ByPermission:
			rec.Included, rec.Basis = true, b
			rec.Summary = "No licence. Included with the maintainer's written permission: " + s.Permission + "."
		case b == ByException:
			rec.Included, rec.Basis, rec.Exception = true, b, ex.find(s.ID)
			rec.Summary = "No explicit redistribution licence located. Included pending permission, under a recorded exception."
		default:
			rec.Included, rec.Basis = true, b
			rec.Summary = "Included in this research bundle only."
		}
		p.Sources = append(p.Sources, rec)
	}
	return p
}

func CheckDeployable(p Provenance, reg *Registry, ex *Exceptions) error {
	if p.Mode != "production" {
		return fmt.Errorf("provenance: bundle %s is a %s bundle and must not be deployed", p.Version, p.Mode)
	}
	var problems []string
	for _, rec := range p.Sources {
		if !rec.Included {
			continue
		}
		i := slices.IndexFunc(reg.Sources, func(s Source) bool { return s.ID == rec.ID })
		switch {
		case i < 0:
			problems = append(problems, rec.ID+" is not a registered source")
		case reg.Sources[i].Status == Redistributable, reg.Sources[i].Status == Permitted:
		case reg.Sources[i].Status == PermissionRequired && ex.find(rec.ID) != nil:
		default:
			problems = append(problems, fmt.Sprintf("%s is %s with no recorded exception", rec.ID, reg.Sources[i].Status))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("provenance: bundle %s includes data it may not ship: %s", p.Version, strings.Join(problems, "; "))
	}
	return nil
}
