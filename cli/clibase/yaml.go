package clibase

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/go-wordwrap"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

var (
	_ yaml.Marshaler   = new(OptionSet)
	_ yaml.Unmarshaler = new(OptionSet)
)

// deepMapNode returns the mapping node at the given path,
// creating it if it doesn't exist.
func deepMapNode(n *yaml.Node, path []string, headComment string) *yaml.Node {
	if len(path) == 0 {
		return n
	}

	// Name is every two nodes.
	for i := 0; i < len(n.Content)-1; i += 2 {
		if n.Content[i].Value == path[0] {
			// Found matching name, recurse.
			return deepMapNode(n.Content[i+1], path[1:], headComment)
		}
	}

	// Not found, create it.
	nameNode := yaml.Node{
		Kind:        yaml.ScalarNode,
		Value:       path[0],
		HeadComment: headComment,
	}
	valueNode := yaml.Node{
		Kind: yaml.MappingNode,
	}
	n.Content = append(n.Content, &nameNode)
	n.Content = append(n.Content, &valueNode)
	return deepMapNode(&valueNode, path[1:], headComment)
}

// MarshalYAML converts the option set to a YAML node, that can be
// converted into bytes via yaml.Marshal.
//
// The node is returned to enable post-processing higher up in
// the stack.
//
// It is isomorphic with FromYAML.
func (s *OptionSet) MarshalYAML() (any, error) {
	root := yaml.Node{
		Kind: yaml.MappingNode,
	}

	for _, opt := range *s {
		if opt.YAML == "" {
			continue
		}

		defValue := opt.Default
		if defValue == "" {
			defValue = "<unset>"
		}
		comment := wordwrap.WrapString(
			fmt.Sprintf("%s (default: %s)", opt.Description, defValue),
			80,
		)
		nameNode := yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       opt.YAML,
			HeadComment: wordwrap.WrapString(comment, 80),
		}
		var valueNode yaml.Node
		if m, ok := opt.Value.(yaml.Marshaler); ok {
			v, err := m.MarshalYAML()
			if err != nil {
				return nil, xerrors.Errorf(
					"marshal %q: %w", opt.Name, err,
				)
			}
			valueNode, ok = v.(yaml.Node)
			if !ok {
				return nil, xerrors.Errorf(
					"marshal %q: unexpected underlying type %T",
					opt.Name, v,
				)
			}
		} else {
			// The all-other types case.
			//
			// A bit of a hack, we marshal and then unmarshal to get
			// the underlying node.
			byt, err := yaml.Marshal(opt.Value)
			if err != nil {
				return nil, xerrors.Errorf(
					"marshal %q: %w", opt.Name, err,
				)
			}

			var docNode yaml.Node
			err = yaml.Unmarshal(byt, &docNode)
			if err != nil {
				return nil, xerrors.Errorf(
					"unmarshal %q: %w", opt.Name, err,
				)
			}
			if len(docNode.Content) != 1 {
				return nil, xerrors.Errorf(
					"unmarshal %q: expected one node, got %d",
					opt.Name, len(docNode.Content),
				)
			}

			valueNode = *docNode.Content[0]
		}
		var group []string
		for _, g := range opt.Group.Ancestry() {
			if g.YAML == "" {
				return nil, xerrors.Errorf(
					"group yaml name is empty for %q, groups: %+v",
					opt.Name,
					opt.Group,
				)
			}
			group = append(group, g.YAML)
		}
		var groupDesc string
		if opt.Group != nil {
			groupDesc = wordwrap.WrapString(opt.Group.Description, 80)
		}
		parentValueNode := deepMapNode(
			&root, group,
			groupDesc,
		)
		parentValueNode.Content = append(
			parentValueNode.Content,
			&nameNode,
			&valueNode,
		)
	}
	return &root, nil
}

// mapYAMLNodes converts n into a map with keys of form "group.subgroup.option"
// and values of the corresponding YAML nodes.
func mapYAMLNodes(parent *yaml.Node) (map[string]*yaml.Node, error) {
	if parent.Kind != yaml.MappingNode {
		return nil, xerrors.Errorf("expected mapping node, got type %v", parent.Kind)
	}
	if len(parent.Content)%2 != 0 {
		return nil, xerrors.Errorf("expected an even number of k/v pairs, got %d", len(parent.Content))
	}
	var (
		key  string
		m    = make(map[string]*yaml.Node)
		merr error
	)
	for i, child := range parent.Content {
		if i%2 == 0 {
			if child.Kind != yaml.ScalarNode {
				return nil, xerrors.Errorf("expected scalar node for key, got type %v", child.Kind)
			}
			key = child.Value
			continue
		}
		// Even if we have a mapping node, we don't know if it's a grouped simple
		// option a complex option, so we store both "key" and "group.key". Since
		// we're storing pointers, the additional memory is of little concern.
		m[key] = child
		if child.Kind == yaml.MappingNode {
			sub, err := mapYAMLNodes(child)
			if err != nil {
				merr = errors.Join(merr, xerrors.Errorf("mapping node %q: %w", key, err))
				continue
			}
			for k, v := range sub {
				m[key+"."+k] = v
			}
		}
	}

	return m, nil
}

func (o *Option) setFromYAMLNode(n *yaml.Node) error {
	o.ValueSource = ValueSourceYAML
	if um, ok := o.Value.(yaml.Unmarshaler); ok {
		return um.UnmarshalYAML(n)
	}

	switch n.Kind {
	case yaml.ScalarNode:
		return o.Value.Set(n.Value)
	case yaml.SequenceNode:
		return n.Decode(o.Value)
	case yaml.MappingNode:
		return xerrors.Errorf("mapping node must implement yaml.Unmarshaler")
	default:
		return xerrors.Errorf("unexpected node kind %v", n.Kind)
	}
}

// UnmarshalYAML converts the given YAML node into the option set.
// It is isomorphic with ToYAML.
func (s *OptionSet) UnmarshalYAML(rootNode *yaml.Node) error {
	// The rootNode will be a DocumentNode if it's read from a file. Currently,
	// we don't support multiple YAML documents.
	if rootNode.Kind == yaml.DocumentNode {
		if len(rootNode.Content) != 1 {
			return xerrors.Errorf("expected one node in document, got %d", len(rootNode.Content))
		}
		rootNode = rootNode.Content[0]
	}

	yamlNodes, err := mapYAMLNodes(rootNode)
	if err != nil {
		return xerrors.Errorf("mapping nodes: %w", err)
	}

	matchedNodes := make(map[string]*yaml.Node, len(yamlNodes))

	var merr error
	for i := range *s {
		opt := &(*s)[i]
		if opt.YAML == "" {
			continue
		}
		var group []string
		for _, g := range opt.Group.Ancestry() {
			if g.YAML == "" {
				return xerrors.Errorf(
					"group yaml name is empty for %q, groups: %+v",
					opt.Name,
					opt.Group,
				)
			}
			group = append(group, g.YAML)
			delete(yamlNodes, strings.Join(group, "."))
		}
		key := strings.Join(append(group, opt.YAML), ".")
		node, ok := yamlNodes[key]
		if !ok {
			continue
		}

		if err := opt.setFromYAMLNode(node); err != nil {
			merr = errors.Join(merr, xerrors.Errorf("setting %q: %w", opt.YAML, err))
		}
		matchedNodes[key] = node
	}

	// Remove all matched nodes and their descendants from yamlNodes so we
	// can accurately report unknown options.
	for k := range yamlNodes {
		var key string
		for _, part := range strings.Split(k, ".") {
			if key != "" {
				key += "."
			}
			key += part
			if _, ok := matchedNodes[key]; ok {
				delete(yamlNodes, k)
			}
		}
	}
	for k := range yamlNodes {
		merr = errors.Join(merr, xerrors.Errorf("unknown option %q", k))
	}

	return merr
}
