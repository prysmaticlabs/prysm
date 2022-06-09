package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/prysmaticlabs/prysm/sszgen"
	"github.com/prysmaticlabs/prysm/sszgen/testutil"
	"github.com/urfave/cli/v2"
)

var ir = &cli.Command{
	Name:    "ir",
	ArgsUsage: "<input package, eg github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1>",
	Aliases: []string{"gen"},
	Usage:   "generate intermediate representation for a go struct type. This data structure is used by the backend code generator. Outputting it to a source file an be useful for generating test cases and debugging.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Value:       "",
			Usage:       "file path to write generated code",
			Destination: &output,
			Required: true,
		},
		&cli.StringFlag{
			Name:        "type-names",
			Value:       "",
			Usage:       "if specified, only generate types specified in this comma-separated list",
			Destination: &typeNames,
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() > 0 {
			sourcePackage = c.Args().Get(0)
		}
		index := sszgen.NewPackageIndex()
		rep := sszgen.NewRepresenter(index)

		var err error
		var specs []*sszgen.DeclarationRef
		if len(typeNames) > 0 {
			for _, n := range strings.Split(strings.TrimSpace(typeNames), ",") {
				specs = append(specs, &sszgen.DeclarationRef{Package: sourcePackage, Name: n})
			}
		} else {
			specs, err = index.DeclarationRefs(sourcePackage)
			if err != nil {
				return err
			}
		}
		if len(specs) == 0 {
			return fmt.Errorf("Could not find any codegen targets in source package %s", sourcePackage)
		}

		outFh, err := os.Create(output)
		defer outFh.Close()
		if err != nil {
			return err
		}

		renderedTypes := make([]string, 0)
		for _, s := range specs {
			typeRep, err := rep.GetDeclaration(s.Package, s.Name)
			if err != nil {
				return err
			}
			rendered, err := testutil.RenderIntermediate(typeRep)
			if err != nil {
				return err
			}
			renderedTypes = append(renderedTypes, rendered)
		}
		if err != nil {
			return err
		}

		_, err = io.Copy(outFh, strings.NewReader(strings.Join(renderedTypes, "\n")))
		return err
	},
}