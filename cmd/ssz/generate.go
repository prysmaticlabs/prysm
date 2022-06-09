package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/prysmaticlabs/prysm/sszgen"
	"github.com/prysmaticlabs/prysm/sszgen/backend"
	"github.com/urfave/cli/v2"
)

var sourcePackage, output, typeNames string
var generate = &cli.Command{
	Name:    "generate",
	ArgsUsage: "<input package, eg github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1>",
	Aliases: []string{"gen"},
	Usage:   "generate methodsets for a go struct type to support ssz ser/des",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Value:       "",
			Usage:       "directory to write generated code (same as input by default)",
			Destination: &output,
		},
		&cli.StringFlag{
			Name:        "type-names",
			Value:       "",
			Usage:       "if specified, only generate methods for types specified in this comma-separated list",
			Destination: &typeNames,
		},
	},
	Action: func(c *cli.Context) error {
		sourcePackage = c.Args().Get(0)
		if sourcePackage == "" {
			cli.ShowCommandHelp(c, "generate")
			return fmt.Errorf("error: mising required <input package> argument")
		}
		var err error
		index := sszgen.NewPackageIndex()
		packageName, err := index.GetPackageName(sourcePackage)
		if err != nil {
			return err
		}
		rep := sszgen.NewRepresenter(index)

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

		if output == "" {
			output = "methodical.ssz.go"
		}
		outFh, err := os.Create(output)
		defer outFh.Close()
		if err != nil {
			return err
		}

		g := backend.NewGenerator(packageName, sourcePackage)
		for _, s := range specs {
			fmt.Printf("Generating methods for %s/%s\n", s.Package, s.Name)
			typeRep, err := rep.GetDeclaration(s.Package, s.Name)
			if err != nil {
				return err
			}
			g.Generate(typeRep)
		}
		rbytes, err := g.Render()
		if err != nil {
			return err
		}
		_, err = io.Copy(outFh, bytes.NewReader(rbytes))
		return err
	},
}