/*
Copyright 2025 The KubeOne Authors.

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

package helmchart

import (
	"fmt"
	"text/template"
	"text/template/parse"
)

// Translate rewrites a KubeOne addon manifest template (rooted at the addon
// render context) into a Helm chart template (rooted at .Values). Only the
// top-level context fields are rewritten; all template functions are left
// untouched and are supplied at render time through Helm's
// CustomTemplateFuncs.
func Translate(src string) (string, error) {
	tpl := template.New("addon").Funcs(TxtFuncMap(""))

	parsed, err := tpl.Parse(src)
	if err != nil {
		return "", err
	}

	tree := parsed.Tree
	if tree == nil {
		return "", fmt.Errorf("parsing addon manifest: no template tree")
	}

	walk(tree.Root)

	return tree.Root.String(), nil
}

func walk(n parse.Node) {
	if n == nil {
		return
	}

	switch node := n.(type) {
	case *parse.ListNode:
		for _, c := range node.Nodes {
			walk(c)
		}
	case *parse.ActionNode:
		walk(node.Pipe)
	case *parse.PipeNode:
		for _, c := range node.Cmds {
			walk(c)
		}
	case *parse.CommandNode:
		rewriteInternalImagesGet(node)

		for _, a := range node.Args {
			walk(a)
		}
	case *parse.FieldNode:
		rewriteField(node)
	case *parse.ChainNode:
		walk(node.Node)
	case *parse.IfNode:
		walkBranch(&node.BranchNode)
	case *parse.RangeNode:
		walkBranch(&node.BranchNode)
	case *parse.WithNode:
		walkBranch(&node.BranchNode)
	case *parse.TemplateNode:
		walk(node.Pipe)
	}
}

func walkBranch(b *parse.BranchNode) {
	walk(b.Pipe)
	walk(b.List)

	if b.ElseList != nil {
		walk(b.ElseList)
	}
}

// rewriteField rewrites a template rooted at .<templateDataField> into
// .Values.<key>.
func rewriteField(node *parse.FieldNode) {
	if node == nil || len(node.Ident) == 0 {
		return
	}

	key, ok := fieldMapping[node.Ident[0]]
	if !ok {
		return
	}

	ident := make([]string, 0, len(node.Ident)+1)
	ident = append(ident, "Values", key)
	ident = append(ident, node.Ident[1:]...)

	node.Ident = ident
}

// rewriteInternalImagesGet rewrites .InternalImages.Get "X" into getImage "X".
// getImage is a custom template function provided by KubeOne at render time.
func rewriteInternalImagesGet(node *parse.CommandNode) {
	if node == nil || len(node.Args) == 0 {
		return
	}

	field, ok := node.Args[0].(*parse.FieldNode)
	if !ok || len(field.Ident) != 2 {
		return
	}

	if field.Ident[0] != "InternalImages" || field.Ident[1] != "Get" {
		return
	}

	node.Args[0] = &parse.IdentifierNode{
		NodeType: parse.NodeIdentifier,
		Ident:    "getImage",
	}
}
