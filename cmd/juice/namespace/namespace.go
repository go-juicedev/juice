package namespace

import (
	"errors"
	"flag"
	"os"
	"strings"

	"github.com/eatmoreapple/juice/cmd/juice/internal"
)

// Generate is a command for generating namespace.
type Generate struct{}

func (n *Generate) Name() string {
	return "namespace"
}

func (n *Generate) Do() error {
	var _type string
	c := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	c.StringVar(&_type, "type", "", "typeName type name")
	_ = c.Parse(os.Args[2:])
	if _type == "" {
		return errors.New("namespace: type is required")
	}
	cmp := &internal.NameSpaceAutoComplete{TypeName: _type}
	namespace, err := cmp.Autocomplete()
	if err != nil {
		return err
	}
	println(namespace)
	return nil
}

func (n *Generate) Help() string {
	var builder strings.Builder
	builder.WriteString("namespace: generate namespace for type\n")
	builder.WriteString("  Usage:\n")
	builder.WriteString("    --type string interface type name\n")
	return builder.String()
}