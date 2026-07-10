/*
Copyright 2026 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package parser

// NodeKind identifies a parsed dynamic SQL node.
type NodeKind uint8

const (
	TextNodeKind NodeKind = iota
	IfNodeKind
	BindNodeKind
	ForeachNodeKind
	ChooseNodeKind
	TrimNodeKind
	WhereNodeKind
	SetNodeKind
	IncludeNodeKind
)

// Node is a format-independent dynamic SQL node.
type Node interface {
	Kind() NodeKind
}

type TextNode struct {
	Text string
}

func (TextNode) Kind() NodeKind { return TextNodeKind }

type IfNode struct {
	Test     string
	Children []Node
}

func (IfNode) Kind() NodeKind { return IfNodeKind }

type BindNode struct {
	Name  string
	Value string
}

func (BindNode) Kind() NodeKind { return BindNodeKind }

type ForeachNode struct {
	Collection string
	Item       string
	Index      string
	Open       string
	Close      string
	Separator  string
	Children   []Node
}

func (ForeachNode) Kind() NodeKind { return ForeachNodeKind }

type WhenNode struct {
	Test     string
	Children []Node
}

type ChooseNode struct {
	Bindings     []BindNode
	Whens        []WhenNode
	Otherwise    []Node
	HasOtherwise bool
}

func (ChooseNode) Kind() NodeKind { return ChooseNodeKind }

type TrimNode struct {
	Prefix          string
	Suffix          string
	PrefixOverrides string
	SuffixOverrides string
	Children        []Node
}

func (TrimNode) Kind() NodeKind { return TrimNodeKind }

type WhereNode struct {
	Children []Node
}

func (WhereNode) Kind() NodeKind { return WhereNodeKind }

type SetNode struct {
	Children []Node
}

func (SetNode) Kind() NodeKind { return SetNodeKind }

type IncludeNode struct {
	RefID      string
	Properties map[string]string
}

func (IncludeNode) Kind() NodeKind { return IncludeNodeKind }
