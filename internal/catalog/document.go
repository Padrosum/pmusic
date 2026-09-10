package catalog

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Document is the GenLang interchange JSON object: types, sets, entities,
// memberships. See The GenLang Book, chapter 14.
type Document struct {
	Types       []Type       `json:"types"`
	Sets        []Set        `json:"sets"`
	Entities    []Entity     `json:"entities"`
	Memberships []Membership `json:"memberships"`
}

type Type struct {
	Kind       string          `json:"kind"`
	Name       string          `json:"name"`
	Parent     *string         `json:"parent"`
	Properties json.RawMessage `json:"properties"`
}

type Set struct {
	Name       string          `json:"name"`
	Properties json.RawMessage `json:"properties"`
}

type Entity struct {
	Name  string          `json:"name"`
	Type  *string         `json:"type"`
	Value json.RawMessage `json:"value"`
}

type Membership struct {
	Entity string `json:"entity"`
	Set    string `json:"set"`
}

func ParseJSON(data []byte) (Document, error) {
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return Document{}, fmt.Errorf("genlang json: %w", err)
	}
	if doc.Types == nil {
		doc.Types = []Type{}
	}
	if doc.Sets == nil {
		doc.Sets = []Set{}
	}
	if doc.Entities == nil {
		doc.Entities = []Entity{}
	}
	if doc.Memberships == nil {
		doc.Memberships = []Membership{}
	}
	return doc, nil
}

func (d Document) entityMap() map[string]Entity {
	out := make(map[string]Entity, len(d.Entities))
	for _, e := range d.Entities {
		out[e.Name] = e
	}
	return out
}

func (d Document) parentOf() map[string]string {
	out := make(map[string]string, len(d.Types))
	for _, t := range d.Types {
		if t.Parent != nil && *t.Parent != "" {
			out[t.Name] = *t.Parent
		}
	}
	return out
}

func (d Document) childrenOf() map[string][]string {
	out := make(map[string][]string)
	for _, t := range d.Types {
		if t.Parent != nil && *t.Parent != "" {
			out[*t.Parent] = append(out[*t.Parent], t.Name)
		}
	}
	return out
}

func (d Document) findType(name string) (Type, bool) {
	for _, t := range d.Types {
		if strings.EqualFold(t.Name, name) {
			return t, true
		}
	}
	return Type{}, false
}

func (d Document) findSet(name string) (Set, bool) {
	for _, s := range d.Sets {
		if strings.EqualFold(s.Name, name) {
			return s, true
		}
	}
	return Set{}, false
}

// Ancestors returns parent types of name, excluding itself, nearest first.
func (d Document) Ancestors(name string) []string {
	t, ok := d.findType(name)
	if !ok {
		return nil
	}
	parentOf := d.parentOf()
	var out []string
	seen := map[string]bool{t.Name: true}
	cur := t.Name
	for {
		p, ok := parentOf[cur]
		if !ok || p == "" || seen[p] {
			break
		}
		seen[p] = true
		out = append(out, p)
		cur = p
	}
	return out
}

// Descendants returns types below name, declaration-adjacent BFS, excluding itself.
func (d Document) Descendants(name string) []string {
	t, ok := d.findType(name)
	if !ok {
		return nil
	}
	children := d.childrenOf()
	var out []string
	seen := map[string]bool{t.Name: true}
	queue := []string{t.Name}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, child := range children[cur] {
			if seen[child] {
				continue
			}
			seen[child] = true
			out = append(out, child)
			queue = append(queue, child)
		}
	}
	return out
}

func (d Document) TypesOf(entityName string) []string {
	ent, ok := d.entityMap()[entityName]
	if !ok || ent.Type == nil || *ent.Type == "" {
		return nil
	}
	out := []string{*ent.Type}
	out = append(out, d.Ancestors(*ent.Type)...)
	return out
}

func (d Document) SetsOf(entityName string) []string {
	var out []string
	for _, m := range d.Memberships {
		if m.Entity == entityName {
			out = append(out, m.Set)
		}
	}
	return out
}

func (d Document) Members(setName string) []string {
	s, ok := d.findSet(setName)
	if !ok {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, m := range d.Memberships {
		if m.Set != s.Name || seen[m.Entity] {
			continue
		}
		seen[m.Entity] = true
		out = append(out, m.Entity)
	}
	return out
}
