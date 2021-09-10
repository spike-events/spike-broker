package postman

import (
	"encoding/json"
	"log"
	urlpkg "net/url"
	"os"
	"strings"
)

type Collection struct {
	Info   Info      `json:"info"`
	Folder []*Folder `json:"item"`
}

func CreateCollection(nome string) *Collection {
	return &Collection{
		Info: Info{
			Name:   nome,
			Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
	}
}

func (c *Collection) Write(file *os.File) error {
	content, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	_, err = file.Write(content)
	return err
}

func (c *Collection) AddItemGroup(nome string) *Folder {
	f := &Folder{
		Name: nome,
	}
	c.Folder = append(c.Folder, f)
	return f
}

type Folder struct {
	Name string  `json:"name"`
	Item []*Item `json:"item"`
}

func (f *Folder) AddItem(bearer *string, name string, req Request, res interface{}) {

	u, err := urlpkg.Parse(req.URL.Raw)
	req.URL.Protocol = u.Scheme
	req.URL.Host = strings.Split(u.Hostname(), ".")
	req.URL.Port = u.Port()
	req.URL.Path = strings.Split(u.Path, "/")

	for key, v := range u.Query() {
		req.URL.Query = append(req.URL.Query, Query{
			Key:   key,
			Value: v[0],
		})
	}

	log.Println(u, err)

	if bearer != nil {
		req.Auth = Auth{
			Type: "bearer",
			Bearer: []Bearer{{
				Key:   "token",
				Value: *bearer,
				Type:  "string",
			}},
		}
	}

	it := &Item{
		Name:    name,
		Request: req,
	}
	if res != nil {
		it.Response = []interface{}{res}
	}
	f.Item = append(f.Item, it)
}

type Info struct {
	PostmanID string `json:"_postman_id"`
	Name      string `json:"name"`
	Schema    string `json:"schema"`
}
type Bearer struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}
type Auth struct {
	Type   string   `json:"type"`
	Bearer []Bearer `json:"bearer"`
}
type Header struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}
type Body struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}
type Query struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type URL struct {
	Raw      string   `json:"raw"`
	Protocol string   `json:"protocol"`
	Host     []string `json:"host"`
	Port     string   `json:"port"`
	Path     []string `json:"path"`
	Query    []Query  `json:"query"`
}
type Request struct {
	Auth   Auth     `json:"auth,omitempty"`
	Method string   `json:"method"`
	Header []Header `json:"header"`
	Body   Body     `json:"body"`
	URL    URL      `json:"url"`
}
type Item struct {
	Name     string        `json:"name"`
	Request  Request       `json:"request"`
	Response []interface{} `json:"response"`
}
