// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/manifoldco/promptui"
)

//History contains the history path, binary name and history items.
type History struct {
	ConfigPath string
	Items      []string
	Item       string
	Prompt     []PromptItem
	PromptItem PromptItem
}

//SetPath set History path
func SetPath(path string) *History {
	return &History{
		ConfigPath: path,
	}
}

//Load the history record
func (h *History) Load() *History {
	f, err := os.Open(h.ConfigPath)
	if err != nil {
		fmt.Println("cannot open the file", err)
		os.Exit(1)
	}
	defer f.Close()
	rd := bufio.NewReader(f)

	var items []string

	for {
		line, _, err := rd.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("Load err:", err)
			os.Exit(1)
		}
		items = append(items, string(line))
	}
	h.Items = items
	return h
}

//List History records order by Ascending
func (h *History) List() *History {
	if len(h.Items) > 1000 {
		temp := make([]string, 1000)
		copy(temp, h.Items[:1000])
		h.Items = temp
		return h
	}
	return h
}

//Reverse History records order by Descending
func (h *History) Reverse() *History {
	h.Items = reverse(h.Items)
	return h
}

//Select one History item from history
func (h *History) Select() *History {
	h.Prompt = PromptItems(h.Items)
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "\U0001F4CC {{ .Cmd | cyan }} ",
		Inactive: "  {{ .Cmd | cyan }}",
		Selected: "\U0001F4CC {{ .Cmd | red | cyan }} ",
		Details: `
--------- Target Info ----------
{{ "Garden:" | faint }}{{ if eq .Garden "live" }}	{{ .Garden | red }}{{ else }}	{{ .Garden }}{{end}}
{{ "Project:" | faint }}{{ if eq .Garden "live" }}	{{ .Project | red }}{{ else }}	{{ .Project }}{{end}}
{{ "Seed:" | faint }}{{ if eq .Garden "live" }}	{{ .Seed | red }}{{ else }}	{{ .Seed }}{{end}}
{{ "Namespace:" | faint }}{{ if eq .Garden "live" }}	{{ .Namespace | red }}{{ else }}	{{ .Namespace }}{{end}}
{{ "Shoot:" | faint }}{{ if eq .Garden "live" }}	{{ .Shoot | red }}{{ else }}	{{ .Shoot }}{{end}}
`,
	}
	searcher := func(input string, index int) bool {
		Promptitem := h.Prompt[index]
		name := strings.Replace(strings.ToLower(Promptitem.Cmd), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)
		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:     "Target history",
		Items:     h.Prompt,
		Templates: templates,
		Size:      10,
		Searcher:  searcher,
	}

	i, _, err := prompt.Run()

	if err != nil {
		fmt.Print("Prompt Exit")
		os.Exit(0)
	}

	h.PromptItem = h.Prompt[i]
	h.Item = h.Items[i]
	return h
}

func reverse(s []string) []string {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
	return s
}

func factory(str string) *PromptItem {
	p := &PromptItem{}
	if err := json.Unmarshal([]byte(str), &p); err != nil {
		log.Fatal(err)
	}
	return p
}

//PromptItem struct
type PromptItem struct {
	Cmd       string `json:"cmd,omitempty"`
	Garden    string `json:"garden,omitempty"`
	Project   string `json:"project,omitempty"`
	Seed      string `json:"seed,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Shoot     string `json:"shoot,omitempty"`
}

//Prompt struct
type Prompt struct {
	Items []PromptItem
}

//PromptItems generate prompt items
func PromptItems(load []string) []PromptItem {
	items := []PromptItem{}
	Prompt := Prompt{items}
	for _, i := range load {
		Prompt.Items = append(Prompt.Items, *factory(i))
	}
	return Prompt.Items
}
